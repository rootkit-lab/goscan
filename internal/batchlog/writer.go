package batchlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"goscan/internal/checker"
	"goscan/internal/paths"
)

var secretRe = regexp.MustCompile(`(?i)((?:password|passwd|secret|token|api[_-]?key|auth)\s*[=:]\s*)([^\s&"']+)`)

type ManifestOpts struct {
	Quick        bool  `json:"quick"`
	Limit        int   `json:"limit"`
	ScriptID     string `json:"scriptId,omitempty"`
	FindingID    int64  `json:"findingId,omitempty"`
	UnopenedOnly bool  `json:"unopenedOnly,omitempty"`
	UntestedOnly bool  `json:"untestedOnly,omitempty"`
	ForceRecheck bool  `json:"forceRecheck,omitempty"`
}

type Manifest struct {
	RunID     string       `json:"runId"`
	StartedAt string       `json:"startedAt"`
	Secs      int          `json:"secs"`
	Threads   int          `json:"threads"`
	Findings  int          `json:"findings"`
	Checks    int          `json:"checks"`
	OK        int          `json:"ok"`
	Fail      int          `json:"fail"`
	Skip      int          `json:"skip"`
	Opts      ManifestOpts `json:"opts"`
}

type CheckRecord struct {
	FindingID  int64  `json:"findingId"`
	Domain     string `json:"domain"`
	ScriptID   string `json:"scriptId"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exitCode"`
	Summary    string `json:"summary"`
	LogPath    string `json:"logPath,omitempty"`
	ErrorClass string `json:"errorClass,omitempty"`
	Ms         int64  `json:"ms"`
}

type StartOpts struct {
	RepoRoot     string
	LogRoot      string // optional; default var/logs/batch under RepoRoot
	RunID        string
	Threads      int
	Findings     int
	Checks       int
	ManifestOpts ManifestOpts
}

// Writer persists batch run logs under var/logs/batch/{runId}/.
type Writer struct {
	runDir   string
	runID    string
	started  time.Time
	manifest Manifest
	mu       sync.Mutex
	summary  *os.File
	results  *os.File
	records  []CheckRecord
}

func NewWriter(opts StartOpts) (*Writer, error) {
	if opts.RunID == "" {
		opts.RunID = paths.NewBatchRunID()
	}
	root := opts.RepoRoot
	if root == "" {
		var err error
		root, err = paths.RepoRoot()
		if err != nil {
			return nil, err
		}
	}
	batchRoot := paths.BatchLogsRoot(root)
	if opts.LogRoot != "" {
		batchRoot = opts.LogRoot
	}
	runDir := filepath.Join(batchRoot, opts.RunID)
	for _, sub := range []string{runDir, filepath.Join(runDir, "by-finding"), filepath.Join(runDir, "failures")} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return nil, err
		}
	}

	summaryPath := filepath.Join(runDir, "summary.txt")
	summary, err := os.OpenFile(summaryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	resultsPath := filepath.Join(runDir, "results.jsonl")
	results, err := os.OpenFile(resultsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		summary.Close()
		return nil, err
	}

	started := time.Now().UTC()
	w := &Writer{
		runDir:  runDir,
		runID:   opts.RunID,
		started: started,
		summary: summary,
		results: results,
		manifest: Manifest{
			RunID:     opts.RunID,
			StartedAt: started.Format(time.RFC3339),
			Threads:   opts.Threads,
			Findings:  opts.Findings,
			Checks:    opts.Checks,
			Opts:      opts.ManifestOpts,
		},
	}
	if err := w.writeManifestPartial(); err != nil {
		w.Close()
		return nil, err
	}
	return w, nil
}

func (w *Writer) RunDir() string { return w.runDir }
func (w *Writer) RunID() string  { return w.runID }

func (w *Writer) AppendSummaryLine(line string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintln(w.summary, line)
}

func (w *Writer) RecordCheck(rec CheckRecord, output string) (logRelPath string, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	domainDir := sanitizeName(rec.Domain)
	logRel := filepath.Join("by-finding", domainDir, rec.ScriptID+".log")
	logAbs := filepath.Join(w.runDir, logRel)
	if err := os.MkdirAll(filepath.Dir(logAbs), 0o755); err != nil {
		return "", err
	}
	body := RedactSecrets(output)
	if err := os.WriteFile(logAbs, []byte(body), 0o644); err != nil {
		return "", err
	}

	rec.LogPath = filepath.ToSlash(logRel)
	if rec.ErrorClass == "" {
		rec.ErrorClass = checker.ClassifyError(rec.ScriptID, rec.Status, rec.Summary, output)
	}

	w.records = append(w.records, rec)
	line, err := json.Marshal(rec)
	if err != nil {
		return rec.LogPath, err
	}
	if _, err := w.results.Write(append(line, '\n')); err != nil {
		return rec.LogPath, err
	}
	return rec.LogPath, nil
}

type FinishStats struct {
	OK   int
	Fail int
	Skip int
}

func (w *Writer) Finish(stats FinishStats, elapsed time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.manifest.OK = stats.OK
	w.manifest.Fail = stats.Fail
	w.manifest.Skip = stats.Skip
	w.manifest.Secs = int(elapsed.Round(time.Second).Seconds())

	if err := w.writeManifestLocked(); err != nil {
		return err
	}
	if err := WriteFailureReports(w.runDir, w.records); err != nil {
		return err
	}
	return w.updateLatestSymlink()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var err1, err2 error
	if w.summary != nil {
		err1 = w.summary.Close()
		w.summary = nil
	}
	if w.results != nil {
		err2 = w.results.Close()
		w.results = nil
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (w *Writer) writeManifestPartial() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeManifestLocked()
}

func (w *Writer) writeManifestLocked() error {
	data, err := json.MarshalIndent(w.manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(w.runDir, "manifest.json"), data, 0o644)
}

func (w *Writer) updateLatestSymlink() error {
	batchRoot := filepath.Dir(w.runDir)
	latest := filepath.Join(batchRoot, "latest")
	_ = os.Remove(latest)
	return os.Symlink(w.runDir, latest)
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}

// RedactSecrets masks common credential patterns in log output.
func RedactSecrets(s string) string {
	return secretRe.ReplaceAllString(s, "${1}***")
}
