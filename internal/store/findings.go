package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Finding represents a saved .env discovery.
type Finding struct {
	ID              int64  `json:"id"`
	Domain          string `json:"domain"`
	Path            string `json:"path"`
	URL             string `json:"url"`
	Confidence      string `json:"confidence"`
	FilePath        string `json:"filePath"`
	ScanRunID       string `json:"scanRunId"`
	FoundAt         string `json:"foundAt"`
	HasCredentials  bool   `json:"hasCredentials"`
	ContentHash     string `json:"contentHash"`
	OpenedAt        string `json:"openedAt"`
}

type FindingsFilter struct {
	Query          string
	Confidence     string
	HasCredentials *bool
	UnopenedOnly   bool
	Limit          int
}

type FindingsStats struct {
	Total    int64 `json:"total"`
	Unopened int64 `json:"unopened"`
}

// FindingsStore manages findings table + FTS5 index.
type FindingsStore struct {
	db          *sql.DB
	findingsDir string
}

func OpenFindingsStore(db *sql.DB, findingsDir string) (*FindingsStore, error) {
	fs := &FindingsStore{db: db, findingsDir: findingsDir}
	if err := fs.migrate(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *FindingsStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS findings (
			id INTEGER PRIMARY KEY,
			domain TEXT NOT NULL,
			path TEXT NOT NULL,
			url TEXT,
			confidence TEXT,
			file_path TEXT NOT NULL,
			scan_run_id TEXT,
			found_at TEXT DEFAULT (datetime('now')),
			has_credentials INTEGER DEFAULT 0,
			content_hash TEXT NOT NULL,
			content TEXT,
			UNIQUE(domain, path, content_hash)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_domain ON findings(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_scan ON findings(scan_run_id)`,
	}
	for _, s := range stmts {
		if _, err := fs.db.Exec(s); err != nil {
			return err
		}
	}
	if err := fs.ensureOpenedAtColumn(); err != nil {
		return err
	}

	var name string
	err := fs.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='findings_fts'`).Scan(&name)
	if err == sql.ErrNoRows {
		_, err = fs.db.Exec(`CREATE VIRTUAL TABLE findings_fts USING fts5(
			domain, path, content,
			content='findings', content_rowid='id'
		)`)
		if err != nil {
			return err
		}
		_, err = fs.db.Exec(`CREATE TRIGGER IF NOT EXISTS findings_ai AFTER INSERT ON findings BEGIN
			INSERT INTO findings_fts(rowid, domain, path, content) VALUES (new.id, new.domain, new.path, new.content);
		END`)
		if err != nil {
			return err
		}
		_, err = fs.db.Exec(`CREATE TRIGGER IF NOT EXISTS findings_ad AFTER DELETE ON findings BEGIN
			INSERT INTO findings_fts(findings_fts, rowid, domain, path, content) VALUES('delete', old.id, old.domain, old.path, old.content);
		END`)
		if err != nil {
			return err
		}
		_, err = fs.db.Exec(`CREATE TRIGGER IF NOT EXISTS findings_au AFTER UPDATE ON findings BEGIN
			INSERT INTO findings_fts(findings_fts, rowid, domain, path, content) VALUES('delete', old.id, old.domain, old.path, old.content);
			INSERT INTO findings_fts(rowid, domain, path, content) VALUES (new.id, new.domain, new.path, new.content);
		END`)
	}
	return err
}

func (fs *FindingsStore) ensureOpenedAtColumn() error {
	var name string
	err := fs.db.QueryRow(`SELECT name FROM pragma_table_info('findings') WHERE name='opened_at'`).Scan(&name)
	if err == sql.ErrNoRows {
		_, err = fs.db.Exec(`ALTER TABLE findings ADD COLUMN opened_at TEXT`)
	}
	return err
}

func (fs *FindingsStore) MarkOpened(id int64) error {
	_, err := fs.db.Exec(`UPDATE findings SET opened_at = datetime('now') WHERE id = ? AND (opened_at IS NULL OR opened_at = '')`, id)
	return err
}

func (fs *FindingsStore) Stats() (FindingsStats, error) {
	var s FindingsStats
	err := fs.db.QueryRow(`SELECT COUNT(*) FROM findings`).Scan(&s.Total)
	if err != nil {
		return s, err
	}
	err = fs.db.QueryRow(`SELECT COUNT(*) FROM findings WHERE opened_at IS NULL OR opened_at = ''`).Scan(&s.Unopened)
	return s, err
}

func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// LabelFromPath converts HTTP path to filename label (inverse of PathFromLabel).
func LabelFromPath(path string) string {
	label := strings.TrimPrefix(path, "/")
	label = strings.NewReplacer("/", "_", ".", "_").Replace(label)
	if label == "" {
		return "root"
	}
	return label
}

// PathFromLabel converts filename label back to HTTP path.
func PathFromLabel(label string) string {
	if label == "" || label == "root" {
		return "/"
	}
	p := strings.ReplaceAll(label, "_", ".")
	if !strings.HasPrefix(p, ".") {
		p = "." + p
	}
	return "/" + p
}

// DomainFileName returns relative path under by-domain/.
func DomainFileName(domain, path string) string {
	label := LabelFromPath(path)
	return filepath.Join(domain, label+".env")
}

// ParseLegacyFilename parses scan_resultados flat filenames.
func ParseLegacyFilename(name string) (domain, path string, ok bool) {
	name = strings.TrimSuffix(name, ".env")
	idx := strings.LastIndex(name, "__")
	if idx < 0 {
		return "", "", false
	}
	domainPart := name[:idx]
	label := name[idx+2:]
	domain = strings.ReplaceAll(domainPart, "_", ".")
	path = PathFromLabel(label)
	return domain, path, true
}

// SaveFinding writes content to by-domain/ and inserts DB row. Returns finding ID and whether it is new.
func (fs *FindingsStore) SaveFinding(domain, path, url, confidence, scanRunID string, content []byte, hasCredentials bool) (int64, string, bool, error) {
	hash := ContentHash(content)
	rel := DomainFileName(domain, path)
	abs := filepath.Join(fs.findingsDir, "by-domain", rel)

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return 0, "", false, err
	}

	var existingID int64
	err := fs.db.QueryRow(
		`SELECT id FROM findings WHERE domain = ? AND path = ? AND content_hash = ?`,
		domain, path, hash,
	).Scan(&existingID)
	if err == nil {
		return existingID, rel, false, nil
	}

	if err := os.WriteFile(abs, content, 0644); err != nil {
		return 0, "", false, err
	}

	hc := 0
	if hasCredentials {
		hc = 1
	}
	res, err := fs.db.Exec(`INSERT INTO findings
		(domain, path, url, confidence, file_path, scan_run_id, has_credentials, content_hash, content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		domain, path, url, confidence, rel, scanRunID, hc, hash, string(content))
	if err != nil {
		return 0, "", false, err
	}
	id, _ := res.LastInsertId()
	return id, rel, true, nil
}

func (fs *FindingsStore) Get(id int64) (*Finding, string, error) {
	var f Finding
	var hc int
	var content string
	err := fs.db.QueryRow(`SELECT id, domain, path, url, confidence, file_path, scan_run_id, found_at, has_credentials, content_hash, content
		FROM findings WHERE id = ?`, id).Scan(
		&f.ID, &f.Domain, &f.Path, &f.URL, &f.Confidence, &f.FilePath, &f.ScanRunID, &f.FoundAt, &hc, &f.ContentHash, &content)
	if err != nil {
		return nil, "", err
	}
	f.HasCredentials = hc == 1
	return &f, content, nil
}

func (fs *FindingsStore) GetFilePath(id int64) (string, error) {
	var rel string
	err := fs.db.QueryRow(`SELECT file_path FROM findings WHERE id = ?`, id).Scan(&rel)
	return rel, err
}

func (fs *FindingsStore) Search(filter FindingsFilter) ([]Finding, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	unopenedFTS := ""
	unopenedPlain := ""
	if filter.UnopenedOnly {
		unopenedFTS = ` AND (f.opened_at IS NULL OR f.opened_at = '')`
		unopenedPlain = ` AND (opened_at IS NULL OR opened_at = '')`
	}

	var rows *sql.Rows
	var err error

	if strings.TrimSpace(filter.Query) != "" {
		q := strings.TrimSpace(filter.Query)
		sqlQuery := `SELECT f.id, f.domain, f.path, f.url, f.confidence, f.file_path, f.scan_run_id, f.found_at, f.has_credentials, f.content_hash, COALESCE(f.opened_at, '')
			FROM findings f
			JOIN findings_fts fts ON f.id = fts.rowid
			WHERE findings_fts MATCH ?` + unopenedFTS + `
			ORDER BY f.found_at DESC LIMIT ?`
		rows, err = fs.db.Query(sqlQuery, q+"*", limit)
	} else {
		sqlQuery := `SELECT id, domain, path, url, confidence, file_path, scan_run_id, found_at, has_credentials, content_hash, COALESCE(opened_at, '')
			FROM findings WHERE 1=1` + unopenedPlain + `
			ORDER BY found_at DESC LIMIT ?`
		rows, err = fs.db.Query(sqlQuery, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Finding
	for rows.Next() {
		var f Finding
		var hc int
		if err := rows.Scan(&f.ID, &f.Domain, &f.Path, &f.URL, &f.Confidence, &f.FilePath, &f.ScanRunID, &f.FoundAt, &hc, &f.ContentHash, &f.OpenedAt); err != nil {
			return nil, err
		}
		f.HasCredentials = hc == 1
		if filter.Confidence != "" && f.Confidence != filter.Confidence {
			continue
		}
		if filter.HasCredentials != nil && f.HasCredentials != *filter.HasCredentials {
			continue
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (fs *FindingsStore) ImportLegacyFile(scanRunID, legacyName string, content []byte, url, confidence string, hasCredentials bool) (moved bool, err error) {
	domain, path, ok := ParseLegacyFilename(legacyName)
	if !ok {
		return false, fmt.Errorf("parse filename: %s", legacyName)
	}
	_, rel, _, err := fs.SaveFinding(domain, path, url, confidence, scanRunID, content, hasCredentials)
	if err != nil {
		return false, err
	}
	_ = rel
	return true, nil
}

func (fs *FindingsStore) Count() (int64, error) {
	var n int64
	err := fs.db.QueryRow(`SELECT COUNT(*) FROM findings`).Scan(&n)
	return n, err
}
