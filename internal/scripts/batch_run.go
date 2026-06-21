package scripts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goscan/internal/batchlog"
	"goscan/internal/checker"
)

type BatchExecOpts struct {
	OnProgress func(BatchProgress)
	Workers    int
	LogWriter  *batchlog.Writer
}

type batchFindingMeta struct {
	index       int
	scriptTotal int
}

func buildFindingMeta(items []BatchItem) map[int64]batchFindingMeta {
	order := []int64{}
	seen := map[int64]bool{}
	counts := map[int64]int{}
	for _, it := range items {
		counts[it.FindingID]++
		if !seen[it.FindingID] {
			seen[it.FindingID] = true
			order = append(order, it.FindingID)
		}
	}
	meta := make(map[int64]batchFindingMeta, len(order))
	for i, id := range order {
		meta[id] = batchFindingMeta{index: i + 1, scriptTotal: counts[id]}
	}
	return meta
}

func applyBatchResult(status string, stats *BatchStats) {
	switch status {
	case "ok":
		stats.OK++
	case "skip":
		stats.Skip++
	default:
		stats.Fail++
	}
}

func emitBatchProgress(opts BatchExecOpts, p BatchProgress, stats BatchStats, completed, checkTotal, threads int) {
	if opts.OnProgress == nil {
		return
	}
	p.CheckIndex = completed
	p.CheckTotal = checkTotal
	p.OkCount = stats.OK
	p.FailCount = stats.Fail
	p.SkipCount = stats.Skip
	p.Threads = threads
	opts.OnProgress(p)
}

func processBatchItem(ctx context.Context, r *Runner, item BatchItem, opts BatchExecOpts) (status, summary, logPath string, exitCode int) {
	started := time.Now()
	res := r.RunBatch(ctx, item.Script.ID, item.EnvPath, nil)
	status = checker.ClassifyStatus(res.ExitCode, res.Output)
	summary = checker.SummarizeOutput(res.Output)
	if summary == "" && res.Err != nil {
		summary = res.Err.Error()
	}
	exitCode = res.ExitCode

	if opts.LogWriter != nil {
		rel, _ := opts.LogWriter.RecordCheck(batchlog.CheckRecord{
			FindingID:  item.FindingID,
			Domain:     item.Domain,
			ScriptID:   item.Script.ID,
			Status:     status,
			ExitCode:   exitCode,
			Summary:    summary,
			ErrorClass: checker.ClassifyError(item.Script.ID, status, summary, res.Output),
			Ms:         time.Since(started).Milliseconds(),
		}, res.Output)
		logPath = rel
	}
	return status, summary, logPath, exitCode
}

// ExecuteBatch runs items sequentially or in parallel when Workers > 1.
func (r *Runner) ExecuteBatch(ctx context.Context, items []BatchItem, opts BatchExecOpts) BatchStats {
	if opts.Workers > 1 {
		return r.executeBatchParallel(ctx, items, opts)
	}
	return r.executeBatchSequential(ctx, items, opts)
}

func (r *Runner) executeBatchSequential(ctx context.Context, items []BatchItem, opts BatchExecOpts) BatchStats {
	stats := BatchStats{Total: len(items)}
	if len(items) == 0 {
		return stats
	}

	findingTotal := countFindings(items)
	meta := buildFindingMeta(items)
	var lastFinding int64 = -1
	scriptInFinding := 0
	completed := 0

	for _, item := range items {
		if ctx.Err() != nil {
			break
		}
		if item.FindingID != lastFinding {
			lastFinding = item.FindingID
			scriptInFinding = 0
		}
		scriptInFinding++
		completed++

		status, summary, logPath, exitCode := processBatchItem(ctx, r, item, opts)
		applyBatchResult(status, &stats)

		fm := meta[item.FindingID]
		line := fmt.Sprintf("[%d/%d] %s — %s … %s", fm.index, findingTotal, item.Domain, item.Script.ID, status)
		if summary != "" {
			line += " (" + summary + ")"
		}

		emitBatchProgress(opts, BatchProgress{
			FindingIndex: fm.index,
			FindingTotal: findingTotal,
			FindingID:    item.FindingID,
			Domain:       item.Domain,
			ScriptIndex:  scriptInFinding,
			ScriptTotal:  fm.scriptTotal,
			ScriptID:     item.Script.ID,
			ScriptLabel:  item.Script.Label,
			Status:       status,
			Summary:      summary,
			ExitCode:     exitCode,
			Line:         line,
			LogPath:      logPath,
		}, stats, completed, len(items), 1)
	}
	return stats
}

func (r *Runner) executeBatchParallel(ctx context.Context, items []BatchItem, opts BatchExecOpts) BatchStats {
	stats := BatchStats{Total: len(items)}
	if len(items) == 0 {
		return stats
	}

	workers := opts.Workers
	if workers < 2 {
		workers = 2
	}
	if workers > 16 {
		workers = 16
	}

	findingTotal := countFindings(items)
	meta := buildFindingMeta(items)
	checkTotal := len(items)

	type job struct {
		item          BatchItem
		scriptInFinding int
	}

	jobs := make(chan job, len(items))
	scriptCount := map[int64]int{}
	for _, item := range items {
		scriptCount[item.FindingID]++
		jobs <- job{item: item, scriptInFinding: scriptCount[item.FindingID]}
	}
	close(jobs)

	var mu sync.Mutex
	completed := 0

	var wg sync.WaitGroup
	worker := func() {
		for j := range jobs {
			if ctx.Err() != nil {
				return
			}
			item := j.item
			status, summary, logPath, exitCode := processBatchItem(ctx, r, item, opts)

			mu.Lock()
			applyBatchResult(status, &stats)
			completed++
			fm := meta[item.FindingID]
			line := fmt.Sprintf("[%d/%d] %s — %s … %s", fm.index, findingTotal, item.Domain, item.Script.ID, status)
			if summary != "" {
				line += " (" + summary + ")"
			}
			emitBatchProgress(opts, BatchProgress{
				FindingIndex: fm.index,
				FindingTotal: findingTotal,
				FindingID:    item.FindingID,
				Domain:       item.Domain,
				ScriptIndex:  j.scriptInFinding,
				ScriptTotal:  fm.scriptTotal,
				ScriptID:     item.Script.ID,
				ScriptLabel:  item.Script.Label,
				Status:       status,
				Summary:      summary,
				ExitCode:     exitCode,
				Line:         line,
				LogPath:      logPath,
			}, stats, completed, checkTotal, workers)
			mu.Unlock()
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker()
		}()
	}
	wg.Wait()
	return stats
}

func FormatBatchDone(stats BatchStats, elapsed time.Duration) string {
	return fmt.Sprintf("batch done — OK %d · FAIL %d · SKIP %d · %d total · %s",
		stats.OK, stats.Fail, stats.Skip, stats.Total, elapsed.Round(time.Second))
}
