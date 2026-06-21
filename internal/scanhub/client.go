package scanhub

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Client connects a remote worker to the hub.
type Client struct {
	token   string
	runID   string
	encrypt bool
	conn    *websocket.Conn
	mu      sync.Mutex
	closed  bool
}

// ClientConfig configures a remote hub connection.
type ClientConfig struct {
	URL      string
	Token    string
	RunID    string
	WorkerID string
	Total    int64
	Encrypt  bool
}

// Dial opens the WebSocket and completes the hello handshake.
func Dial(ctx context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("hub url em falta")
	}
	conn, _, err := websocket.Dial(ctx, cfg.URL, nil)
	if err != nil {
		return nil, err
	}
	c := &Client{
		token: cfg.Token, runID: cfg.RunID, encrypt: cfg.Encrypt, conn: conn,
	}
	hello := Message{
		Type: TypeHello, RunID: cfg.RunID, WorkerID: cfg.WorkerID,
		Token: cfg.Token, Total: cfg.Total,
	}
	if err := c.writePlain(ctx, hello); err != nil {
		conn.Close(websocket.StatusInternalError, "")
		return nil, err
	}
	readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for {
		typ, data, err := conn.Read(readCtx)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "")
			return nil, fmt.Errorf("hub hello: %w", err)
		}
		if typ != websocket.MessageText && typ != websocket.MessageBinary {
			continue
		}
		plain := data
		if cfg.Encrypt {
			if p, err := Decrypt(cfg.Token, data); err == nil {
				plain = p
			}
		}
		msg, err := Decode(plain)
		if err != nil {
			continue
		}
		switch msg.Type {
		case TypePong:
			go c.drain()
			return c, nil
		case TypeError:
			conn.Close(websocket.StatusPolicyViolation, msg.Error)
			return nil, fmt.Errorf("hub: %s", msg.Error)
		}
	}
}

func (c *Client) drain() {
	for {
		if _, _, err := c.conn.Read(context.Background()); err != nil {
			return
		}
	}
}

func (c *Client) writePlain(ctx context.Context, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("hub fechado")
	}
	raw, err := Encode(msg)
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, raw)
}

func (c *Client) write(ctx context.Context, msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("hub fechado")
	}
	raw, err := Encode(msg)
	if err != nil {
		return err
	}
	if c.encrypt {
		raw, err = Encrypt(c.token, raw)
		if err != nil {
			return err
		}
	}
	return c.conn.Write(ctx, websocket.MessageText, raw)
}

// SendProgress emits a progress frame.
func (c *Client) SendProgress(ctx context.Context, scanned, vulns, total, rate int64) error {
	return c.write(ctx, Message{
		Type: TypeProgress, Scanned: scanned, Vulns: vulns, Total: total, Rate: rate,
	})
}

// SendFound emits finding metadata and full content.
func (c *Client) SendFound(ctx context.Context, domain, path, url, confidence string, hasCredentials bool, content []byte) error {
	if len(content) > maxContentBytes {
		content = content[:maxContentBytes]
	}
	msg := Message{
		Type: TypeFound, RunID: c.runID, Domain: domain, Path: path, URL: url,
		Confidence: confidence, HasCredentials: hasCredentials,
	}
	if len(content) > 1*1024*1024 {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write(content)
		_ = gz.Close()
		content = buf.Bytes()
		msg.Compressed = true
	}
	msg.ContentB64 = base64.StdEncoding.EncodeToString(content)
	return c.write(ctx, msg)
}

// SendDone signals batch completion.
func (c *Client) SendDone(ctx context.Context, scanned, vulns int64, ok bool, errMsg string) error {
	return c.write(ctx, Message{
		Type: TypeDone, Scanned: scanned, Vulns: vulns, OK: ok, Error: errMsg,
	})
}

// Close closes the WebSocket.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
}
