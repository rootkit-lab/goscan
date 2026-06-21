package remotefindings

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExportJSON writes NDJSON findings filtered by scanRunID (empty = all).
func ExportJSON(findingsDir, scanRunID string, w io.Writer) error {
	path := filepath.Join(findingsDir, "findings.ndjson")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 256*1024)
	sc.Buffer(buf, 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec FindingExport
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return err
		}
		if scanRunID != "" && rec.ScanRunID != scanRunID {
			continue
		}
		out, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(out, '\n')); err != nil {
			return err
		}
	}
	return sc.Err()
}
