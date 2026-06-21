package scanhub

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const maxContentBytes = 50 * 1024 * 1024

// FoundEvent carries a remote finding with full .env content.
type FoundEvent struct {
	WorkerID       string
	RunID          string
	Domain         string
	Path           string
	URL            string
	Confidence     string
	HasCredentials bool
	Content        []byte
}

// ProgressEvent is a scan progress update.
type ProgressEvent struct {
	WorkerID string
	Scanned  int64
	Vulns    int64
	Total    int64
	Rate     int64
}

// DoneEvent signals worker completion.
type DoneEvent struct {
	WorkerID string
	Scanned  int64
	Vulns    int64
	OK       bool
	Error    string
}

// Handlers receives hub events on the orchestrator side.
type Handlers struct {
	OnProgress func(ProgressEvent)
	OnFound    func(FoundEvent)
	OnDone     func(DoneEvent)
}

type workerSession struct {
	workerID string
	runID    string
	token    string
	total    int64
	helloOK  bool
}

// Registry runs the local hub for one orchestrated scan.
type Registry struct {
	mu       sync.Mutex
	srv      *http.Server
	ln       net.Listener
	addr     string
	sessions map[string]*workerSession
	pending  map[string]*foundMeta
	h        Handlers
	closed   bool
}

type foundMeta struct {
	workerID       string
	runID          string
	domain         string
	path           string
	url            string
	confidence     string
	hasCredentials bool
}

// StartRegistry listens on 127.0.0.1:0 and serves WebSocket /hub.
func StartRegistry(ctx context.Context, h Handlers) (*Registry, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	reg := &Registry{
		ln:       ln,
		addr:     ln.Addr().String(),
		sessions: make(map[string]*workerSession),
		pending:  make(map[string]*foundMeta),
		h:        h,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/hub", reg.handleHub)
	reg.srv = &http.Server{Handler: mux}
	go func() { _ = reg.srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		reg.Close()
	}()
	return reg, nil
}

func (r *Registry) LocalAddr() string { return r.addr }

// HubWSURL returns the WebSocket URL for a tunneled remote connection.
func (r *Registry) HubWSURL(remoteHostPort string) string {
	return "ws://" + remoteHostPort + "/hub"
}

// RegisterWorker returns auth token for a remote worker connection.
func (r *Registry) RegisterWorker(workerID, runID string, total int64) (token string, err error) {
	b := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return "", fmt.Errorf("hub fechado")
	}
	r.sessions[token] = &workerSession{
		workerID: workerID, runID: runID, token: token, total: total,
	}
	return token, nil
}

func (r *Registry) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.srv.Shutdown(ctx)
}

func (r *Registry) handleHub(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := req.Context()
	var sess *workerSession
	const encrypt = true

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		plain := data
		if sess != nil && sess.helloOK && encrypt {
			if p, err := Decrypt(sess.token, data); err == nil {
				plain = p
			}
		}
		msg, err := Decode(plain)
		if err != nil {
			continue
		}

		switch msg.Type {
		case TypeHello:
			r.mu.Lock()
			s, ok := r.sessions[msg.Token]
			if !ok || msg.WorkerID != s.workerID {
				r.mu.Unlock()
				_ = r.writeConn(ctx, conn, nil, false, Message{Type: TypeError, Error: "auth inválida"})
				return
			}
			s.helloOK = true
			sess = s
			r.mu.Unlock()
			_ = r.writeConn(ctx, conn, sess, encrypt, Message{Type: TypePong})
		case TypeProgress:
			if sess == nil || !sess.helloOK || r.h.OnProgress == nil {
				continue
			}
			r.h.OnProgress(ProgressEvent{
				WorkerID: sess.workerID, Scanned: msg.Scanned, Vulns: msg.Vulns,
				Total: msg.Total, Rate: msg.Rate,
			})
		case TypeFound:
			if sess == nil || !sess.helloOK {
				continue
			}
			if msg.ContentB64 == "" {
				continue
			}
			body, err := base64.StdEncoding.DecodeString(msg.ContentB64)
			if err != nil || len(body) > maxContentBytes {
				continue
			}
			if msg.Compressed {
				br, gzErr := gzip.NewReader(bytes.NewReader(body))
				if gzErr != nil {
					continue
				}
				body, err = ioutil.ReadAll(br)
				_ = br.Close()
				if err != nil || len(body) > maxContentBytes {
					continue
				}
			}
			r.deliverFound(sess, msg, body)
		case TypeDone:
			if sess != nil && r.h.OnDone != nil {
				r.h.OnDone(DoneEvent{
					WorkerID: sess.workerID, Scanned: msg.Scanned, Vulns: msg.Vulns,
					OK: msg.OK, Error: msg.Error,
				})
			}
			return
		case TypePing:
			_ = r.writeConn(ctx, conn, sess, encrypt, Message{Type: TypePong})
		}
	}
}

func (r *Registry) deliverFound(sess *workerSession, msg Message, body []byte) {
	if r.h.OnFound == nil {
		return
	}
	runID := msg.RunID
	if runID == "" {
		runID = sess.runID
	}
	r.h.OnFound(FoundEvent{
		WorkerID: sess.workerID, RunID: runID,
		Domain: msg.Domain, Path: msg.Path, URL: msg.URL,
		Confidence: msg.Confidence, HasCredentials: msg.HasCredentials,
		Content: body,
	})
}

func (r *Registry) writeConn(ctx context.Context, conn *websocket.Conn, sess *workerSession, encrypt bool, msg Message) error {
	raw, err := Encode(msg)
	if err != nil {
		return err
	}
	if encrypt && sess != nil && sess.helloOK {
		raw, err = Encrypt(sess.token, raw)
		if err != nil {
			return err
		}
	}
	return conn.Write(ctx, websocket.MessageText, raw)
}
