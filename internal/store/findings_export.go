package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	data = sanitizeNDJSONExport(data)
	if len(data) == 0 {
		return 0, nil
	}

	imported := 0
	sc := bufio.NewScanner(bytes.NewReader(data))
	buf := make([]byte, 0, 256*1024)
	sc.Buffer(buf, 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var f FindingExport
		if err := json.Unmarshal(line, &f); err != nil {
			return imported, fmt.Errorf("linha NDJSON inválida (importadas %d): %w", imported, err)
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
	if err := sc.Err(); err != nil {
		return imported, err
	}
	return imported, nil
}

func sanitizeNDJSONExport(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte{0}, nil)
	return bytes.TrimSpace(data)
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
