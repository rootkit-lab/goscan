//go:build !nosqlite

package scanner

import "goscan/internal/store"

type sqliteFindings struct{ fs *store.FindingsStore }

func (s *sqliteFindings) SaveFinding(domain, path, url, confidence, scanRunID string, content []byte, hasCredentials bool) (string, error) {
	_, rel, _, err := s.fs.SaveFinding(domain, path, url, confidence, scanRunID, content, hasCredentials)
	return rel, err
}

func (s *sqliteFindings) DomainFileName(domain, path string) string {
	return store.DomainFileName(domain, path)
}

func setSQLiteFindings(fs *store.FindingsStore) {
	activeFindings = &sqliteFindings{fs: fs}
}
