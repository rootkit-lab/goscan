package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type CheckerResult struct {
	ID        int64
	FindingID int64
	ScriptID  string
	Status    string
	ExitCode  int
	Summary   string
	TestedAt  string
	RunID     string
	LogPath   string
}

type CheckerRun struct {
	RunID     string
	StartedAt string
	OK        int
	Fail      int
	Skip      int
	LogDir    string
}

type CheckerResultsStore struct {
	db *sql.DB
}

func OpenCheckerResultsStore(db *sql.DB) (*CheckerResultsStore, error) {
	s := &CheckerResultsStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *CheckerResultsStore) migrate() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS checker_results (
		id INTEGER PRIMARY KEY,
		finding_id INTEGER NOT NULL,
		script_id TEXT NOT NULL,
		status TEXT NOT NULL,
		exit_code INTEGER NOT NULL DEFAULT 0,
		summary TEXT NOT NULL DEFAULT '',
		tested_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(finding_id, script_id)
	)`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_checker_results_finding ON checker_results(finding_id)`)
	if err != nil {
		return err
	}
	if err := addColumnIfMissing(s.db, "checker_results", "run_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumnIfMissing(s.db, "checker_results", "log_path", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err = s.db.Exec(`CREATE TABLE IF NOT EXISTS checker_runs (
		run_id TEXT PRIMARY KEY,
		started_at TEXT NOT NULL,
		ok INTEGER NOT NULL DEFAULT 0,
		fail INTEGER NOT NULL DEFAULT 0,
		skip INTEGER NOT NULL DEFAULT 0,
		log_dir TEXT NOT NULL DEFAULT ''
	)`)
	return err
}

func addColumnIfMissing(db *sql.DB, table, col, def string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == col {
			return nil
		}
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, def))
	return err
}

func (s *CheckerResultsStore) Save(findingID int64, scriptID, status string, exitCode int, summary, runID, logPath string) error {
	summary = strings.TrimSpace(summary)
	if len(summary) > 500 {
		summary = summary[:500] + "…"
	}
	_, err := s.db.Exec(`INSERT INTO checker_results (finding_id, script_id, status, exit_code, summary, tested_at, run_id, log_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(finding_id, script_id) DO UPDATE SET
			status=excluded.status,
			exit_code=excluded.exit_code,
			summary=excluded.summary,
			tested_at=excluded.tested_at,
			run_id=excluded.run_id,
			log_path=excluded.log_path`,
		findingID, scriptID, status, exitCode, summary, time.Now().UTC().Format(time.RFC3339), runID, logPath)
	return err
}

func (s *CheckerResultsStore) SaveRun(run CheckerRun) error {
	_, err := s.db.Exec(`INSERT INTO checker_runs (run_id, started_at, ok, fail, skip, log_dir)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			ok=excluded.ok,
			fail=excluded.fail,
			skip=excluded.skip,
			log_dir=excluded.log_dir`,
		run.RunID, run.StartedAt, run.OK, run.Fail, run.Skip, run.LogDir)
	return err
}

func (s *CheckerResultsStore) ListByFinding(findingID int64) ([]CheckerResult, error) {
	rows, err := s.db.Query(`SELECT id, finding_id, script_id, status, exit_code, summary, tested_at, run_id, log_path
		FROM checker_results WHERE finding_id = ? ORDER BY tested_at DESC`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckerRows(rows)
}

func (s *CheckerResultsStore) ListByFindings(findingIDs []int64) (map[int64][]CheckerResult, error) {
	out := make(map[int64][]CheckerResult)
	if len(findingIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(findingIDs))
	args := make([]any, len(findingIDs))
	for i, id := range findingIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf(`SELECT id, finding_id, script_id, status, exit_code, summary, tested_at, run_id, log_path
		FROM checker_results WHERE finding_id IN (%s) ORDER BY finding_id, tested_at DESC`,
		strings.Join(placeholders, ","))
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanCheckerRowFromRows(rows)
		if err != nil {
			return nil, err
		}
		out[r.FindingID] = append(out[r.FindingID], *r)
	}
	return out, rows.Err()
}

func scanCheckerRows(rows *sql.Rows) ([]CheckerResult, error) {
	var list []CheckerResult
	for rows.Next() {
		r, err := scanCheckerRowFromRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *r)
	}
	return list, rows.Err()
}

func scanCheckerRowFromRows(rows *sql.Rows) (*CheckerResult, error) {
	var r CheckerResult
	err := rows.Scan(&r.ID, &r.FindingID, &r.ScriptID, &r.Status, &r.ExitCode, &r.Summary, &r.TestedAt, &r.RunID, &r.LogPath)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
