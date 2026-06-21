//go:build !nosqlite

package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goscan/internal/paths"
	"goscan/internal/store"
)

// Run executes ingest and optional vulnerability scan (local master with SQLite).
func Run(ctx context.Context, cfg *Config) error {
	log.SetOutput(io.Discard)
	initHTTPClient(cfg.Timeout)

	if cfg.RunID == "" {
		cfg.RunID = time.Now().Format("20060102_150405")
	}
	runID = cfg.RunID

	findingsDir := cfg.FindingsDir
	if findingsDir == "" && cfg.RepoRoot != "" {
		findingsDir = paths.FindingsRoot(cfg.RepoRoot)
	}
	if findingsDir == "" {
		findingsDir = "var/findings"
	}
	if err := os.MkdirAll(filepath.Join(findingsDir, "by-domain"), 0755); err != nil {
		return fmt.Errorf("criar diretório findings: %w", err)
	}

	printBanner()
	appLog.Printf("📁 Findings: %s (run %s)\n", findingsDir, runID)

	domainStore, err := store.OpenDomainStore(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("abrir SQLite: %w", err)
	}
	defer domainStore.Close()

	fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
	if err != nil {
		return fmt.Errorf("findings store: %w", err)
	}
	setSQLiteFindings(fs)
	currentCfg = cfg

	appLog.Printf("💾 %s — %d total | %d scaneados | %d pendentes\n",
		cfg.DBPath, domainStore.Count(), domainStore.ScannedCount(), domainStore.PendingCount())

	appLog.Println("📁 Escaneando .env / listas .txt...")
	files := findInputFiles(cfg.Dir)
	if len(files) == 0 {
		return fmt.Errorf("nenhum arquivo de entrada encontrado (.env ou .txt)")
	}
	appLog.Printf("✅ %d arquivo(s) encontrado(s)\n", len(files))

	stats := &Stats{StartTime: time.Now()}

	if cfg.OnProgress != nil {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					cfg.OnProgress(*stats)
				}
			}
		}()
	}

	if cfg.ScanVulns {
		printSeparator()
		appLog.Println("🔬 INGESTÃO + SCAN (novos + pendentes, skip scaneados)")
		printSeparator()
		runIngestAndScan(ctx, files, cfg, domainStore, stats)
	} else {
		appLog.Printf("\n🔍 Ingestão de domínios (sem scan)...\n")
		runIngestOnly(ctx, files, domainStore, stats)
	}

	domainStore.Flush()

	total := domainStore.Count()
	if total == 0 && atomic.LoadInt64(&stats.DomainsNew) == 0 {
		appLog.Println("❌ Nenhum domínio válido encontrado")
		if len(files) > 0 {
			debugFile(files[0])
		}
		return nil
	}

	exportDir := filepath.Join(findingsDir, "exports")
	if atomic.LoadInt64(&stats.DomainsNew) > 0 {
		_ = os.MkdirAll(exportDir, 0755)
		if err := domainStore.ExportTXT(filepath.Join(exportDir, "dominios_unicos.txt")); err != nil {
			appLog.Printf("⚠️  Erro ao exportar dominios_unicos.txt: %v", err)
		}
	} else {
		appLog.Println("ℹ️  Nenhum domínio novo — export .txt ignorado")
	}

	printIngestStats(stats, total)
	printFinalSummary(findingsDir, cfg.SaveContent)
	return nil
}

func processDomain(domain, source string, ds *store.DomainStore, stats *Stats, scanMode, rescan bool) bool {
	if ds.Exists(domain) {
		if scanMode {
			if ds.IsScanned(domain) && !rescan {
				atomic.AddInt64(&stats.DomainsScanSkip, 1)
				return false
			}
			atomic.AddInt64(&stats.DomainsPending, 1)
			return true
		}
		atomic.AddInt64(&stats.DomainsSkipped, 1)
		return false
	}
	if ds.InsertIfNew(domain, source) {
		atomic.AddInt64(&stats.DomainsNew, 1)
		atomic.AddInt64(&stats.DomainsFound, 1)
		return scanMode
	}
	atomic.AddInt64(&stats.DomainsSkipped, 1)
	return false
}

func runIngestOnly(ctx context.Context, files []string, ds *store.DomainStore, stats *Stats) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go reportIngestProgress(ctx, stats)

	for _, path := range files {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if isDomainListFile(path) {
			ingestDomainList(ctx, path, ds, stats, nil, false)
		} else {
			ingestEnvFile(ctx, path, ds, stats, nil, false)
		}
		atomic.AddInt64(&stats.FilesProcessed, 1)
	}
	fmt.Fprintln(os.Stderr)
}

func runIngestAndScan(ctx context.Context, files []string, cfg *Config, domainStore *store.DomainStore, stats *Stats) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	scanQueue := make(chan string, cfg.Threads*4)
	results := make(chan VulnResult, 1000)

	var saveWg sync.WaitGroup
	saveWg.Add(1)
	go func() {
		defer saveWg.Done()
		logVulnResults(results)
	}()

	scannedCh := make(chan string, cfg.Threads*4)

	var batchWg sync.WaitGroup
	batchWg.Add(1)
	go func() {
		defer batchWg.Done()
		domainStore.RunScannedBatcher(scannedCh)
	}()

	pathList := criticalPaths
	if cfg.Fast {
		pathList = priorityPaths
	}

	var scanWg sync.WaitGroup
	for i := 0; i < cfg.Threads; i++ {
		scanWg.Add(1)
		go func() {
			defer scanWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case domain, ok := <-scanQueue:
					if !ok {
						return
					}
					found := scanDomain(domain, pathList, cfg.PathWorkers, results, cfg.SaveContent)
					if found > 0 {
						atomic.AddInt64(&stats.VulnsFound, int64(found))
					}
					select {
					case scannedCh <- domain:
					case <-ctx.Done():
						return
					}
					atomic.AddInt64(&stats.DomainsScanned, 1)
				}
			}
		}()
	}

	go reportPipelineProgress(ctx, stats)

	for _, path := range files {
		select {
		case <-ctx.Done():
			close(scanQueue)
			scanWg.Wait()
			close(scannedCh)
			batchWg.Wait()
			close(results)
			saveWg.Wait()
			return
		default:
		}
		if isDomainListFile(path) {
			ingestDomainList(ctx, path, domainStore, stats, scanQueue, cfg.Rescan)
		} else {
			ingestEnvFile(ctx, path, domainStore, stats, scanQueue, cfg.Rescan)
		}
		if ctx.Err() != nil {
			close(scanQueue)
			scanWg.Wait()
			close(scannedCh)
			batchWg.Wait()
			close(results)
			saveWg.Wait()
			return
		}
		atomic.AddInt64(&stats.FilesProcessed, 1)
	}

	close(scanQueue)
	scanWg.Wait()
	close(scannedCh)
	batchWg.Wait()
	close(results)
	saveWg.Wait()
	fmt.Fprintln(os.Stderr)
}

func ingestDomainList(ctx context.Context, path string, ds *store.DomainStore, stats *Stats, scanQueue chan<- string, rescan bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		if ctx.Err() != nil {
			return
		}
		atomic.AddInt64(&stats.LinesProcessed, 1)
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		d := cleanDomain(line)
		if !isValidDomain(d) {
			continue
		}
		scanMode := scanQueue != nil
		if processDomain(d, path, ds, stats, scanMode, rescan) && scanQueue != nil {
			select {
			case scanQueue <- d:
			case <-ctx.Done():
				return
			}
		}
	}
}

func ingestEnvFile(ctx context.Context, path string, ds *store.DomainStore, stats *Stats, scanQueue chan<- string, rescan bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	seen := make(map[string]bool)

	add := func(d string) {
		if ctx.Err() != nil {
			return
		}
		if !isValidDomain(d) || seen[d] {
			return
		}
		seen[d] = true
		scanMode := scanQueue != nil
		if processDomain(d, path, ds, stats, scanMode, rescan) && scanQueue != nil {
			select {
			case scanQueue <- d:
			case <-ctx.Done():
				return
			}
		}
	}

	for _, url := range reURL.FindAllString(content, -1) {
		if domain := extractDomain(url); domain != "" {
			add(domain)
		}
	}
	for _, d := range reDomain.FindAllString(content, -1) {
		add(cleanDomain(d))
	}
	for _, match := range reEmail.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			add(cleanDomain(match[1]))
		}
	}
}
