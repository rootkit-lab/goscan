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

	"goscan/internal/paths"
	"goscan/internal/scanner"
	"goscan/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "findings" {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		runFindingsCLI()
		return
	}
	runScanCLI()
}

func repoRoot() string {
	root, err := paths.RepoRoot()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return root
}

func runScanCLI() {
	root := repoRoot()
	cfg := &scanner.Config{RepoRoot: root}

	flag.StringVar(&cfg.Dir, "dir", filepath.Join(root, "files"), "Diretório com .env ou listas .txt")
	flag.StringVar(&cfg.DBPath, "db", paths.DefaultDBPath(root), "SQLite de domínios")
	flag.StringVar(&cfg.FindingsDir, "findings", paths.FindingsRoot(root), "Diretório de findings")
	flag.StringVar(&cfg.RunID, "run-id", "", "ID do run (default: timestamp)")
	flag.IntVar(&cfg.Threads, "threads", 100, "Workers de scan")
	flag.IntVar(&cfg.PathWorkers, "path-workers", 8, "Paths paralelos por domínio")
	flag.BoolVar(&cfg.Fast, "fast", false, "Só paths .env prioritários")
	flag.BoolVar(&cfg.Rescan, "rescan", false, "Reescanear domínios já scaneados")
	flag.BoolVar(&cfg.ScanVulns, "vuln", true, "Escanear vulnerabilidades")
	flag.BoolVar(&cfg.SaveContent, "save", true, "Salvar conteúdo .env")
	timeout := flag.Int("timeout", 8, "Timeout HTTP (s)")
	flag.Parse()
	cfg.Timeout = time.Duration(*timeout) * time.Second

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := scanner.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func runFindingsCLI() {
	root := repoRoot()
	dbPath := paths.DefaultDBPath(root)
	findingsDir := paths.FindingsRoot(root)
	query := flag.String("query", "", "Pesquisa FTS (domínio/conteúdo)")
	confidence := flag.String("confidence", "", "Filtrar HIGH/MEDIUM/LOW")
	limit := flag.Int("limit", 50, "Limite de resultados")
	flag.Parse()

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

	results, err := fs.Search(store.FindingsFilter{
		Query:      *query,
		Confidence: *confidence,
		Limit:      *limit,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	for _, f := range results {
		creds := ""
		if f.HasCredentials {
			creds = " [creds]"
		}
		fmt.Printf("%d\t%s\t%s\t%s%s\t%s\n", f.ID, f.Domain, f.Path, f.Confidence, creds, f.FoundAt)
	}
	if len(results) == 0 {
		fmt.Println("(nenhum resultado)")
	}
}
