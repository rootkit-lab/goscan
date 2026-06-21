//go:build nosqlite

package workerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goscan/internal/paths"
	"goscan/internal/remotefindings"
	"goscan/internal/scanner"
)

// Server exposes a minimal HTTP worker API (stateless, no SQLite).
type Server struct {
	Token       string
	AppRoot     string
	DataRoot    string
	DBPath      string
	FindingsDir string

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	runID   string
	stats   scanner.Stats
}

type scanRequest struct {
	Dir         string `json:"dir"`
	RunID       string `json:"runId"`
	Threads     int    `json:"threads"`
	PathWorkers int    `json:"pathWorkers"`
	Fast        bool   `json:"fast"`
	TimeoutSec  int    `json:"timeoutSec"`
	Ephemeral   bool   `json:"ephemeral"`
}

// ListenAndServe starts the worker HTTP API.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/scan", s.handleScan)
	mux.HandleFunc("/v1/progress", s.handleProgress)
	mux.HandleFunc("/v1/findings/export", s.handleExport)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
}

func (s *Server) auth(r *http.Request) bool {
	if s.Token == "" {
		return true
	}
	h := r.Header.Get("Authorization")
	return strings.TrimPrefix(h, "Bearer ") == s.Token
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, "scan em execução", http.StatusConflict)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.running = true
	s.cancel = cancel
	s.runID = req.RunID
	s.stats = scanner.Stats{}
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.cancel = nil
			s.mu.Unlock()
		}()
		timeout := req.TimeoutSec
		if timeout <= 0 {
			timeout = 8
		}
		findingsDir := s.FindingsDir
		if req.Ephemeral {
			base := filepath.Join("/tmp", "goscan-run-"+req.RunID)
			findingsDir = filepath.Join(base, "findings")
			defer func() { _ = os.RemoveAll(base) }()
		}
		cfg := &scanner.Config{
			Dir:         req.Dir,
			FindingsDir: findingsDir,
			RunID:       req.RunID,
			Threads:     req.Threads,
			PathWorkers: req.PathWorkers,
			Fast:        req.Fast,
			ScanVulns:   true,
			SaveContent: true,
			Timeout:     time.Duration(timeout) * time.Second,
			OnProgress: func(st scanner.Stats) {
				s.mu.Lock()
				s.stats = st
				s.mu.Unlock()
			},
		}
		stats, err := scanner.RunStateless(ctx, cfg)
		if err == nil {
			s.mu.Lock()
			s.stats = stats
			s.mu.Unlock()
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"running":        s.running,
		"runId":          s.runID,
		"domainsScanned": s.stats.DomainsScanned,
		"vulnsFound":     s.stats.VulnsFound,
		"domainsPending": s.stats.DomainsPending,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	runID := r.URL.Query().Get("runId")
	findingsDir := s.FindingsDir
	if runID != "" {
		findingsDir = filepath.Join("/tmp", "goscan-run-"+runID, "findings")
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	if err := remotefindings.ExportJSON(findingsDir, runID, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// NewFromRoots builds a server with default worker paths.
func NewFromRoots(token string) (*Server, error) {
	appRoot, err := paths.AppRoot()
	if err != nil {
		return nil, err
	}
	dataRoot, err := paths.DataRoot()
	if err != nil {
		return nil, err
	}
	return &Server{
		Token:       token,
		AppRoot:     appRoot,
		DataRoot:    dataRoot,
		DBPath:      paths.DefaultDBPath(dataRoot),
		FindingsDir: paths.FindingsRoot(dataRoot),
	}, nil
}

// Cancel stops the active scan if any.
func (s *Server) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// AddrHelp returns a human-readable listen address description.
func AddrHelp(port int) string {
	return fmt.Sprintf(":%d", port)
}
