package store

import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

// FindingExport is a portable finding record for remote merge.
type FindingExport struct {
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	URL            string `json:"url"`
	Confidence     string `json:"confidence"`
	ScanRunID      string `json:"scanRunId"`
	HasCredentials bool   `json:"hasCredentials"`
	Content        string `json:"content"`
	WorkerID       string `json:"workerId,omitempty"`
}

// ExportFindingsJSON writes findings as NDJSON.
func (fs *FindingsStore) ExportFindingsJSON(w interface{ Write([]byte) (int, error) }, scanRunID string) error {
	q := `SELECT domain, path, url, confidence, scan_run_id, has_credentials, content FROM findings`
	args := []any{}
	if scanRunID != "" {
		q += ` WHERE scan_run_id = ?`
		args = append(args, scanRunID)
	}
	rows, err := fs.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var f FindingExport
		var cred int
		if err := rows.Scan(&f.Domain, &f.Path, &f.URL, &f.Confidence, &f.ScanRunID, &cred, &f.Content); err != nil {
			return err
		}
		f.HasCredentials = cred != 0
		b, err := json.Marshal(f)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ImportFindingsJSON merges NDJSON exports into the local master store.
func (fs *FindingsStore) ImportFindingsJSON(r io.Reader, workerID, masterRunID string) (int, error) {
	data, err := readAll(r)
	if err != nil {
		return 0, err
	}
	imported := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var f FindingExport
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			return imported, err
		}
		if workerID != "" && f.WorkerID == "" {
			f.WorkerID = workerID
		}
		runID := f.ScanRunID
		if masterRunID != "" {
			runID = masterRunID + "-" + workerID
		} else if runID == "" {
			runID = "import"
		}
		_, _, _, err := fs.SaveFinding(f.Domain, f.Path, f.URL, f.Confidence, runID, []byte(f.Content), f.HasCredentials)
		if err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// ExportFindingsJSONFile writes to path.
func (fs *FindingsStore) ExportFindingsJSONFile(path, scanRunID string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fs.ExportFindingsJSON(f, scanRunID)
}
