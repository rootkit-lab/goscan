package scripts

import (
	"context"
	"time"
)

var emailScripts = map[string]bool{
	"chk-smtp": true, "chk-sendgrid": true, "chk-mailgun": true, "chk-twilio": true,
}

var heavyScripts = map[string]bool{
	"chk-mysql": true, "chk-postgres": true, "chk-redis": true, "chk-mongodb": true, "chk-memcached": true,
}

// IsQuickScript excludes email senders and heavy DB introspection in --quick batch.
func IsQuickScript(id string) bool {
	return !emailScripts[id] && !heavyScripts[id]
}

func batchTimeout(scriptID string) time.Duration {
	if emailScripts[scriptID] {
		return 90 * time.Second
	}
	if heavyScripts[scriptID] {
		return 75 * time.Second
	}
	return 45 * time.Second
}

// BatchItem is one finding × script execution unit.
type BatchItem struct {
	FindingID int64
	Domain    string
	EnvPath   string
	Script    ScriptInfo
}

// BatchProgress event payload.
type BatchProgress struct {
	FindingIndex int    `json:"findingIndex"`
	FindingTotal int    `json:"findingTotal"`
	FindingID    int64  `json:"findingId"`
	Domain       string `json:"domain"`
	ScriptIndex  int    `json:"scriptIndex"`
	ScriptTotal  int    `json:"scriptTotal"`
	ScriptID     string `json:"scriptId"`
	ScriptLabel  string `json:"scriptLabel"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	ExitCode     int    `json:"exitCode"`
	Line         string `json:"line"`
	CheckIndex   int    `json:"checkIndex"`
	CheckTotal   int    `json:"checkTotal"`
	OkCount      int    `json:"okCount"`
	FailCount    int    `json:"failCount"`
	SkipCount    int    `json:"skipCount"`
	Threads      int    `json:"threads"`
	LogPath      string `json:"logPath"`
}

// BatchDone event payload.
type BatchDone struct {
	OK    int `json:"ok"`
	Fail  int `json:"fail"`
	Skip  int `json:"skip"`
	Total int `json:"total"`
	Secs  int `json:"secs"`
}

// BatchStats aggregates a batch run.
type BatchStats struct {
	OK    int
	Fail  int
	Skip  int
	Total int
}

// BatchPlanOpts filters compatible script pairs.
type BatchPlanOpts struct {
	ScriptID  string
	ScriptIDs []string // from --filter (empty = all compatible)
	Quick     bool
}

// PlanBatch builds the execution queue for one finding.
func (r *Runner) PlanBatch(envPath, domain string, findingID int64, opts BatchPlanOpts) ([]BatchItem, error) {
	compat, err := r.CompatibleScripts(envPath)
	if err != nil {
		return nil, err
	}
	var items []BatchItem
	for _, s := range compat {
		if !scriptAllowed(s.ID, opts) {
			continue
		}
		if opts.Quick && !IsQuickScript(s.ID) {
			continue
		}
		items = append(items, BatchItem{
			FindingID: findingID,
			Domain:    domain,
			EnvPath:   envPath,
			Script:    s,
		})
	}
	return items, nil
}

func countFindings(items []BatchItem) int {
	seen := map[int64]bool{}
	for _, it := range items {
		seen[it.FindingID] = true
	}
	return len(seen)
}

// RunBatch executes one checker in batch mode (--batch, GOSCAN_BATCH=1).
func (r *Runner) RunBatch(ctx context.Context, scriptID, envPath string, emit EventEmitter) RunResult {
	return r.runScript(ctx, scriptID, envPath, emit, batchTimeout(scriptID), true)
}
