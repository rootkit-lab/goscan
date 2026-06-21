package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"goscan/internal/batchlog"
	"goscan/internal/paths"
	"goscan/internal/scripts"
	"goscan/internal/store"
)

func runTestAllCLI() {
	appRoot, dataRoot := resolveRoots()
	dbPath := paths.DefaultDBPath(dataRoot)
	findingsDir := paths.FindingsRoot(dataRoot)

	unopenedOnly := flag.Bool("unopened-only", false, "Só findings nunca abertos")
	findingID := flag.Int64("finding-id", 0, "Testar um finding específico")
	scriptID := flag.String("script", "", "Só este checker (ex. chk-smtp)")
	filter := flag.String("filter", "", "Filtrar checkers (mysql, db, email, smtp, …)")
	quick := flag.Bool("quick", false, "Excluir email e DB pesados")
	limit := flag.Int("limit", 0, "Máximo de findings (0=todos)")
	threads := flag.Int("threads", 1, "Workers paralelos (1=sequencial, max 16)")
	noLog := flag.Bool("no-log", false, "Não gravar logs em var/logs/batch/")
	logDir := flag.String("log-dir", "", "Directório base de logs (default: var/logs/batch)")
	flag.Parse()

	if *filter != "" && *scriptID != "" {
		fmt.Fprintf(os.Stderr, "❌ use --filter ou --script, não ambos\n")
		os.Exit(2)
	}

	var filterScriptIDs []string
	if *filter != "" {
		ids, err := scripts.ResolveBatchFilter(*filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(2)
		}
		filterScriptIDs = ids
	}

	domainStore, err := store.OpenDomainStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	defer domainStore.Close()

	fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	runner, err := scripts.NewRunner(appRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	checkers, err := store.OpenCheckerResultsStore(domainStore.DB())
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	var findings []store.Finding
	if *findingID > 0 {
		f, _, err := fs.Get(*findingID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ finding %d: %v\n", *findingID, err)
			os.Exit(1)
		}
		findings = []store.Finding{*f}
	} else {
		findings, err = fs.Search(store.FindingsFilter{
			UnopenedOnly: *unopenedOnly,
			Limit:        100000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		if *limit > 0 && len(findings) > *limit {
			findings = findings[:*limit]
		}
	}

	var items []scripts.BatchItem
	planOpts := scripts.BatchPlanOpts{ScriptID: *scriptID, ScriptIDs: filterScriptIDs, Quick: *quick}
	for _, f := range findings {
		abs := filepath.Join(findingsDir, "by-domain", f.FilePath)
		part, err := runner.PlanBatch(abs, f.Domain, f.ID, planOpts)
		if err != nil {
			continue
		}
		items = append(items, part...)
	}

	if len(items) == 0 {
		fmt.Println("(nenhum checker compatível)")
		os.Exit(0)
	}

	workers := batchThreadsCLI(*threads)
	startMsg := fmt.Sprintf("batch start — %d findings · %d checks · %d threads", len(findings), len(items), workers)
	fmt.Printf("[goscan] %s\n", startMsg)
	start := time.Now()

	var logWriter *batchlog.Writer
	runID := paths.NewBatchRunID()
	if !*noLog {
		logRoot := paths.BatchLogsRoot(dataRoot)
		if *logDir != "" {
			logRoot = *logDir
		}
		logWriter, err = batchlog.NewWriter(batchlog.StartOpts{
			RepoRoot: dataRoot,
			LogRoot:  logRoot,
			RunID:    runID,
			Threads:  workers,
			Findings: len(findings),
			Checks:   len(items),
			ManifestOpts: batchlog.ManifestOpts{
				Quick:        *quick,
				Limit:        *limit,
				ScriptID:     *scriptID,
				FindingID:    *findingID,
				UnopenedOnly: *unopenedOnly,
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ logs: %v\n", err)
		} else {
			logWriter.AppendSummaryLine(startMsg)
			fmt.Printf("[goscan] logs → %s\n", logWriter.RunDir())
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	stats := runner.ExecuteBatch(ctx, items, scripts.BatchExecOpts{
		Workers:   workers,
		LogWriter: logWriter,
		OnProgress: func(p scripts.BatchProgress) {
			fmt.Println("[goscan]", p.Line)
			if logWriter != nil {
				logWriter.AppendSummaryLine(p.Line)
			}
			_ = checkers.Save(p.FindingID, p.ScriptID, p.Status, p.ExitCode, p.Summary, runID, p.LogPath)
		},
	})

	elapsed := time.Since(start)
	doneLine := scripts.FormatBatchDone(stats, elapsed)
	fmt.Println("[goscan]", doneLine)

	if logWriter != nil {
		logWriter.AppendSummaryLine(doneLine)
		_ = logWriter.Finish(batchlog.FinishStats{
			OK: stats.OK, Fail: stats.Fail, Skip: stats.Skip,
		}, elapsed)
		_ = checkers.SaveRun(store.CheckerRun{
			RunID:     runID,
			StartedAt: start.UTC().Format(time.RFC3339),
			OK:        stats.OK,
			Fail:      stats.Fail,
			Skip:      stats.Skip,
			LogDir:    logWriter.RunDir(),
		})
		_ = logWriter.Close()
	}

	if stats.Fail > 0 {
		os.Exit(1)
	}
}

func batchThreadsCLI(n int) int {
	if n <= 1 {
		return 1
	}
	if n > 16 {
		return 16
	}
	return n
}
