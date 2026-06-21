//go:build nosqlite

// Binário remoto stateless: scan batch + export NDJSON + worker API (sem SQLite).
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"goscan/internal/paths"
	"goscan/internal/remotefindings"
	"goscan/internal/scanhub"
	"goscan/internal/scanner"
	"goscan/internal/workerapi"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "findings":
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
			if len(os.Args) > 1 && os.Args[1] == "export-json" {
				os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
				runFindingsExportJSON()
				return
			}
			fmt.Fprintf(os.Stderr, "❌ subcomando findings não suportado no worker remoto\n")
			os.Exit(1)
		case "worker":
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
			runWorkerCLI()
			return
		}
	}
	runScanCLI()
}

func resolveRoots() (appRoot, dataRoot string) {
	app, err := paths.AppRoot()
	if err != nil {
		wd, _ := os.Getwd()
		app = wd
	}
	data, err := paths.DataRoot()
	if err != nil {
		data = app
	}
	return app, data
}

func runScanCLI() {
	_, dataRoot := resolveRoots()
	cfg := &scanner.Config{RepoRoot: dataRoot}

	flag.StringVar(&cfg.Dir, "dir", filepath.Join(dataRoot, "files"), "Diretório com listas .txt ou .env")
	flag.StringVar(&cfg.FindingsDir, "findings", paths.FindingsRoot(dataRoot), "Diretório de findings (NDJSON)")
	flag.StringVar(&cfg.RunID, "run-id", "", "ID do run")
	flag.IntVar(&cfg.Threads, "threads", 100, "Workers de scan")
	flag.IntVar(&cfg.PathWorkers, "path-workers", 8, "Paths paralelos por domínio")
	flag.BoolVar(&cfg.Fast, "fast", false, "Só paths .env prioritários")
	flag.BoolVar(&cfg.ScanVulns, "vuln", true, "Escanear vulnerabilidades")
	flag.BoolVar(&cfg.SaveContent, "save", true, "Salvar conteúdo .env")
	batchSize := flag.Int("batch-size", 0, "Total de domínios no batch (progresso remoto)")
	progressJSON := flag.Bool("progress-json", false, "Emitir linhas @goscan/progress no stderr")
	hubURL := flag.String("hub", "", "WebSocket hub URL (ws://127.0.0.1:PORT/hub)")
	hubToken := flag.String("hub-token", "", "Token de autenticação do hub")
	workerID := flag.String("worker-id", "", "ID do worker no hub")
	timeout := flag.Int("timeout", 8, "Timeout HTTP (s)")
	showVersion := flag.Bool("version", false, "Mostrar versão")
	flag.Parse()
	if *showVersion {
		appRoot, _ := resolveRoots()
		fmt.Println(paths.InstallVersion(appRoot))
		return
	}
	cfg.Timeout = time.Duration(*timeout) * time.Second
	cfg.BatchSize = *batchSize
	cfg.ProgressJSON = *progressJSON

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var hubClient *scanhub.Client
	if *hubURL != "" && *hubToken != "" {
		c, err := scanhub.Dial(ctx, scanhub.ClientConfig{
			URL: *hubURL, Token: *hubToken, RunID: cfg.RunID,
			WorkerID: *workerID, Total: int64(*batchSize), Encrypt: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ hub indisponível (%v) — fallback stderr\n", err)
		} else {
			hubClient = c
			defer hubClient.Close()
			fmt.Fprintf(os.Stderr, "@goscan/hub connected\n")
			cfg.HubReporter = func(domain, path, url, confidence string, content []byte, hasCredentials bool) {
				_ = hubClient.SendFound(context.Background(), domain, path, url, confidence, hasCredentials, content)
			}
			cfg.HubProgress = func(scanned, vulns, total, rate int64) {
				_ = hubClient.SendProgress(context.Background(), scanned, vulns, total, rate)
			}
		}
	}

	if err := scanner.RunStateless(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		if hubClient != nil {
			_ = hubClient.SendDone(context.Background(), 0, 0, false, err.Error())
		}
		os.Exit(1)
	}
	if hubClient != nil {
		_ = hubClient.SendDone(context.Background(), int64(*batchSize), 0, true, "")
	}
}

func runFindingsExportJSON() {
	_, dataRoot := resolveRoots()
	findingsDir := flag.String("findings", paths.FindingsRoot(dataRoot), "Diretório de findings")
	runID := flag.String("run-id", "", "Filtrar por scan run id")
	flag.Parse()

	if err := remotefindings.ExportJSON(*findingsDir, *runID, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func runWorkerCLI() {
	listen := flag.String("listen", ":9090", "Endereço HTTP")
	token := flag.String("token", "", "Token Bearer (gerado se vazio)")
	flag.Parse()

	tok := *token
	if tok == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		tok = hex.EncodeToString(b)
		fmt.Fprintf(os.Stderr, "worker token: %s\n", tok)
	}

	srv, err := workerapi.NewFromRoots(tok)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "🌐 worker API em %s\n", *listen)
	if err := srv.ListenAndServe(*listen); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
