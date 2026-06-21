package remotefindings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FindingExport matches store.FindingExport for master import.
type FindingExport struct {
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	URL            string `json:"url"`
	Confidence     string `json:"confidence"`
	ScanRunID      string `json:"scanRunId"`
	HasCredentials bool   `json:"hasCredentials"`
	Content        string `json:"content"`
}

// Store persists findings as files + NDJSON (no SQLite).
type Store struct {
	dir        string
	ndjsonPath string
	mu         sync.Mutex
	seen       map[string]struct{}
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("findings dir em falta")
	}
	if err := os.MkdirAll(filepath.Join(dir, "by-domain"), 0755); err != nil {
		return nil, err
	}
	return &Store{
		dir:        dir,
		ndjsonPath: filepath.Join(dir, "findings.ndjson"),
		seen:       make(map[string]struct{}),
	}, nil
}

func contentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func labelFromPath(path string) string {
	label := strings.TrimPrefix(path, "/")
	label = strings.NewReplacer("/", "_", ".", "_").Replace(label)
	if label == "" {
		return "root"
	}
	return label
}

// DomainFileName returns relative path under by-domain/.
func DomainFileName(domain, path string) string {
	return filepath.Join(domain, labelFromPath(path)+".env")
}

// SaveFinding writes content file and appends NDJSON record.
func (s *Store) SaveFinding(domain, path, url, confidence, scanRunID string, content []byte, hasCredentials bool) (rel string, err error) {
	hash := contentHash(content)
	key := domain + "\x00" + path + "\x00" + hash
	s.mu.Lock()
	if _, ok := s.seen[key]; ok {
		s.mu.Unlock()
		return DomainFileName(domain, path), nil
	}
	s.seen[key] = struct{}{}
	s.mu.Unlock()

	rel = DomainFileName(domain, path)
	abs := filepath.Join(s.dir, "by-domain", rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, content, 0644); err != nil {
		return "", err
	}

	rec := FindingExport{
		Domain: domain, Path: path, URL: url, Confidence: confidence,
		ScanRunID: scanRunID, HasCredentials: hasCredentials, Content: string(content),
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return rel, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.ndjsonPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return rel, err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return rel, err
}
