package store

import (
	"bufio"
	"database/sql"
	"os"
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
	mu    sync.RWMutex
	cache map[string]*domainState
	count int64
	batch [][2]string
}

func OpenDomainStore(path string) (*DomainStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
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
