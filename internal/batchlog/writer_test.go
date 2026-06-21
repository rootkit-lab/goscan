package batchlog_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"goscan/internal/batchlog"
)

func TestWriterCreatesRunArtifacts(t *testing.T) {
	root := t.TempDir()
	w, err := batchlog.NewWriter(batchlog.StartOpts{
		RepoRoot: root,
		RunID:    "test_run",
		Threads:  2,
		Findings: 1,
		Checks:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	w.AppendSummaryLine("line 1")
	rel, err := w.RecordCheck(batchlog.CheckRecord{
		FindingID: 1, Domain: "example.com", ScriptID: "chk-smtp",
		Status: "fail", ExitCode: 1, Summary: "535 auth", ErrorClass: "auth", Ms: 100,
	}, "Falha SMTP: 535 auth failed\n")
	if err != nil {
		t.Fatal(err)
	}
	if rel == "" {
		t.Fatal("expected log path")
	}
	if err := w.Finish(batchlog.FinishStats{Fail: 1}, time.Second); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"manifest.json", "summary.txt", "results.jsonl", "failures/smtp-top-errors.txt"} {
		if _, err := os.Stat(filepath.Join(w.RunDir(), name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
