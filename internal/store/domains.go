package store

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const InsertBatchSize = 1000

type domainState struct {
	scanned bool
}

// DomainStore persiste domínios únicos em SQLite com cache em memória.
type DomainStore struct {
	db    *sql.DB
	dbMu  sync.Mutex // serializa transacções (MaxOpenConns=1)
	mu    sync.RWMutex
	cache map[string]*domainState
	count int64
	batch [][2]string
}

func OpenDomainStore(path string) (*DomainStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS domains (
		domain TEXT PRIMARY KEY NOT NULL,
		first_seen TEXT NOT NULL DEFAULT (datetime('now')),
		source_file TEXT,
		scanned_at TEXT
	)`); err != nil {
		db.Close()
		return nil, err
	}

	db.Exec(`ALTER TABLE domains ADD COLUMN scanned_at TEXT`)

	s := &DomainStore{db: db, cache: make(map[string]*domainState)}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_domains_pending ON domains(scanned_at) WHERE scanned_at IS NULL`); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.loadCache(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *DomainStore) DB() *sql.DB {
	return s.db
}

func (s *DomainStore) Close() error {
	s.Flush()
	return s.db.Close()
}

func (s *DomainStore) loadCache() error {
	rows, err := s.db.Query(`SELECT domain, scanned_at FROM domains`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var n int64
	for rows.Next() {
		var domain string
		var scannedAt sql.NullString
		if err := rows.Scan(&domain, &scannedAt); err != nil {
			return err
		}
		s.cache[domain] = &domainState{scanned: scannedAt.Valid}
		n++
	}
	atomic.StoreInt64(&s.count, n)
	return rows.Err()
}

func (s *DomainStore) Exists(domain string) bool {
	s.mu.RLock()
	_, ok := s.cache[domain]
	s.mu.RUnlock()
	return ok
}

func (s *DomainStore) IsScanned(domain string) bool {
	s.mu.RLock()
	st, ok := s.cache[domain]
	s.mu.RUnlock()
	return ok && st.scanned
}

func (s *DomainStore) ScannedCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var n int64
	for _, st := range s.cache {
		if st.scanned {
			n++
		}
	}
	return n
}

func (s *DomainStore) PendingCount() int64 {
	var n int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM domains WHERE scanned_at IS NULL`).Scan(&n); err == nil {
		return n
	}
	return s.Count() - s.ScannedCount()
}

func (s *DomainStore) InsertIfNew(domain, sourceFile string) bool {
	s.mu.Lock()
	if _, ok := s.cache[domain]; ok {
		s.mu.Unlock()
		return false
	}
	s.cache[domain] = &domainState{}
	s.batch = append(s.batch, [2]string{domain, sourceFile})
	atomic.AddInt64(&s.count, 1)
	needFlush := len(s.batch) >= InsertBatchSize
	s.mu.Unlock()

	if needFlush {
		s.Flush()
	}
	return true
}

func (s *DomainStore) Flush() error {
	s.mu.Lock()
	if len(s.batch) == 0 {
		s.mu.Unlock()
		return nil
	}
	batch := s.batch
	s.batch = nil
	s.mu.Unlock()

	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO domains (domain, source_file) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, pair := range batch {
		if _, err := stmt.Exec(pair[0], pair[1]); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ImportBatch inserts many new domains in one flush cycle (optimized for bulk ingest).
func (s *DomainStore) ImportBatch(domains []string, sourceFile string) (int, error) {
	if len(domains) == 0 {
		return 0, nil
	}
	s.mu.Lock()
	added := 0
	for _, domain := range domains {
		if domain == "" {
			continue
		}
		if _, ok := s.cache[domain]; ok {
			continue
		}
		s.cache[domain] = &domainState{}
		s.batch = append(s.batch, [2]string{domain, sourceFile})
		added++
	}
	if added > 0 {
		atomic.AddInt64(&s.count, int64(added))
	}
	s.mu.Unlock()
	if added == 0 {
		return 0, nil
	}
	return added, s.Flush()
}

func (s *DomainStore) MarkScanned(domain string) {
	s.mu.Lock()
	if st, ok := s.cache[domain]; ok {
		st.scanned = true
	}
	s.mu.Unlock()
}

func (s *DomainStore) RunScannedBatcher(ch <-chan string) {
	batch := make([]string, 0, 500)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		tx, err := s.db.Begin()
		if err != nil {
			return
		}
		stmt, err := tx.Prepare(`UPDATE domains SET scanned_at = datetime('now') WHERE domain = ?`)
		if err != nil {
			tx.Rollback()
			return
		}
		for _, d := range batch {
			stmt.Exec(d)
		}
		stmt.Close()
		tx.Commit()
		batch = batch[:0]
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case d, ok := <-ch:
			if !ok {
				flush()
				return
			}
			s.MarkScanned(d)
			batch = append(batch, d)
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *DomainStore) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

func (s *DomainStore) ExportTXT(path string) error {
	rows, err := s.db.Query(`SELECT domain FROM domains ORDER BY domain`)
	if err != nil {
		return err
	}
	defer rows.Close()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 4*1024*1024)
	defer writer.Flush()

	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return err
		}
		if _, err := writer.WriteString(d + "\n"); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountPending returns domains eligible for scan (pending or all if rescan).
func (s *DomainStore) CountPending(rescan bool) int64 {
	if rescan {
		return s.Count()
	}
	return s.PendingCount()
}

// FetchWorkerChunk returns up to limit pending domains for a stable worker partition (rowid % workerCount).
func (s *DomainStore) FetchWorkerChunk(workerIndex, workerCount, limit int, rescan bool) ([]string, error) {
	if workerCount <= 0 || limit <= 0 {
		return nil, nil
	}
	if workerIndex < 0 || workerIndex >= workerCount {
		return nil, fmt.Errorf("workerIndex %d fora de [0,%d)", workerIndex, workerCount)
	}
	var q string
	if rescan {
		q = `SELECT domain FROM domains WHERE (rowid % ?) = ? ORDER BY rowid LIMIT ?`
	} else {
		q = `SELECT domain FROM domains WHERE scanned_at IS NULL AND (rowid % ?) = ? ORDER BY rowid LIMIT ?`
	}
	rows, err := s.db.Query(q, workerCount, workerIndex, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, limit)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// FetchPendingLimit returns up to limit pending domains from the central store (ordered).
func (s *DomainStore) FetchPendingLimit(limit int, rescan bool) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	var q string
	if rescan {
		q = `SELECT domain FROM domains ORDER BY rowid LIMIT ?`
	} else {
		q = `SELECT domain FROM domains WHERE scanned_at IS NULL ORDER BY rowid LIMIT ?`
	}
	rows, err := s.db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, limit)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkScannedList marks specific domains as scanned in the central store.
func (s *DomainStore) MarkScannedList(domains []string) error {
	const batchSize = 500
	for i := 0; i < len(domains); i += batchSize {
		end := i + batchSize
		if end > len(domains) {
			end = len(domains)
		}
		batch := domains[i:end]
		if err := s.markScannedBatch(batch); err != nil {
			return err
		}
	}
	return nil
}

func (s *DomainStore) markScannedBatch(batch []string) error {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * 40 * time.Millisecond)
		}
		lastErr = s.markScannedBatchOnce(batch)
		if lastErr == nil {
			return nil
		}
		if !sqliteBusy(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func sqliteBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "sqlite_busy_snapshot")
}

func (s *DomainStore) markScannedBatchOnce(batch []string) error {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`UPDATE domains SET scanned_at = datetime('now') WHERE domain = ? AND scanned_at IS NULL`)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, d := range batch {
		if _, err := stmt.Exec(d); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		return err
	}
	s.mu.Lock()
	for _, d := range batch {
		if st, ok := s.cache[d]; ok {
			st.scanned = true
		}
	}
	s.mu.Unlock()
	return nil
}

// workerPartitionSubquery assigns pending domains round-robin by order (not rowid parity).
func workerPartitionSubquery(rescan bool) string {
	if rescan {
		return `SELECT domain, (ROW_NUMBER() OVER (ORDER BY rowid) - 1) AS ord FROM domains`
	}
	return `SELECT domain, (ROW_NUMBER() OVER (ORDER BY rowid) - 1) AS ord FROM domains WHERE scanned_at IS NULL`
}

// CountWorkerChunk returns how many domains a worker partition has (stable rowid assignment).
func (s *DomainStore) CountWorkerChunk(workerIndex, workerCount int, rescan bool) int64 {
	if workerCount <= 0 {
		return 0
	}
	var q string
	if rescan {
		q = `SELECT COUNT(*) FROM domains WHERE (rowid % ?) = ?`
	} else {
		q = `SELECT COUNT(*) FROM domains WHERE scanned_at IS NULL AND (rowid % ?) = ?`
	}
	var n int64
	if err := s.db.QueryRow(q, workerCount, workerIndex).Scan(&n); err != nil {
		return 0
	}
	return n
}

// WriteWorkerChunkDir streams a worker partition from SQLite into chunk/domains.txt.
func (s *DomainStore) WriteWorkerChunkDir(ctx context.Context, workerIndex, workerCount int, rescan bool) (dir string, cleanup func(), count int, err error) {
	if workerCount <= 0 {
		return "", nil, 0, fmt.Errorf("workerCount inválido")
	}
	dir, err = os.MkdirTemp("", "goscan-chunk-*")
	if err != nil {
		return "", nil, 0, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	path := filepath.Join(dir, "domains.txt")
	f, err := os.Create(path)
	if err != nil {
		cleanup()
		return "", nil, 0, err
	}
	w := bufio.NewWriterSize(f, 256*1024)

	q := `SELECT domain FROM (` + workerPartitionSubquery(rescan) + `) WHERE (ord % ?) = ?`
	rows, err := s.db.Query(q, workerCount, workerIndex)
	if err != nil {
		f.Close()
		cleanup()
		return "", nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			f.Close()
			cleanup()
			return "", nil, count, ctx.Err()
		}
		var d string
		if err := rows.Scan(&d); err != nil {
			f.Close()
			cleanup()
			return "", nil, 0, err
		}
		if _, err := w.WriteString(d + "\n"); err != nil {
			f.Close()
			cleanup()
			return "", nil, 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		f.Close()
		cleanup()
		return "", nil, 0, err
	}
	if err := w.Flush(); err != nil {
		f.Close()
		cleanup()
		return "", nil, 0, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, 0, err
	}
	return dir, cleanup, count, nil
}

// MarkScannedForWorker marks all domains in a worker partition as scanned.
func (s *DomainStore) MarkScannedForWorker(workerIndex, workerCount int, rescan bool) error {
	q := `UPDATE domains SET scanned_at = datetime('now') WHERE domain IN (
		SELECT domain FROM (` + workerPartitionSubquery(rescan) + `) WHERE (ord % ?) = ?
	)`
	if _, err := s.db.Exec(q, workerCount, workerIndex); err != nil {
		return err
	}
	rows, err := s.db.Query(`SELECT domain FROM (`+workerPartitionSubquery(rescan)+`) WHERE (ord % ?) = ?`, workerCount, workerIndex)
	if err != nil {
		return err
	}
	defer rows.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return err
		}
		if st, ok := s.cache[d]; ok {
			st.scanned = true
		}
	}
	return rows.Err()
}
