//go:build !nosqlite

package scanner

import (
	"bufio"
	"context"
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

// RunChunkScan HTTP-scans domains listed in cfg.Dir (chunk file) without ingest or domain marking.
// The orchestrator marks scanned_at after the chunk completes.
func RunChunkScan(ctx context.Context, cfg *Config) error {
	log.SetOutput(io.Discard)
	initHTTPClient(cfg.Timeout)

	if cfg.RunID == "" {
		cfg.RunID = time.Now().Format("20060102_150405")
	}
	runID = cfg.RunID
	currentCfg = cfg

	findingsDir := cfg.FindingsDir
	if findingsDir == "" && cfg.RepoRoot != "" {
		findingsDir = paths.FindingsRoot(cfg.RepoRoot)
	}
	if findingsDir == "" {
		findingsDir = "var/findings"
	}
	if err := os.MkdirAll(filepath.Join(findingsDir, "by-domain"), 0755); err != nil {
		return err
	}

	if fs, ok := cfg.FindingsStore.(*store.FindingsStore); ok && fs != nil {
		setSQLiteFindings(fs)
	} else {
		domainStore, err := store.OpenDomainStore(cfg.DBPath)
		if err != nil {
			return err
		}
		defer domainStore.Close()

		fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
		if err != nil {
			return err
		}
		setSQLiteFindings(fs)
	}

	files := findInputFiles(cfg.Dir)
	if len(files) == 0 {
		return nil
	}

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

	runChunkScan(ctx, files, cfg, stats)
	return ctx.Err()
}

func runChunkScan(ctx context.Context, files []string, cfg *Config, stats *Stats) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	threads := cfg.Threads
	if threads <= 0 {
		threads = 50
	}
	pathWorkers := cfg.PathWorkers
	if pathWorkers <= 0 {
		pathWorkers = 8
	}

	scanQueue := make(chan string, threads*4)
	results := make(chan VulnResult, 1000)

	var saveWg sync.WaitGroup
	saveWg.Add(1)
	go func() {
		defer saveWg.Done()
		logVulnResults(results)
	}()

	pathList := criticalPaths
	if cfg.Fast {
		pathList = priorityPaths
	}

	var scanWg sync.WaitGroup
	for i := 0; i < threads; i++ {
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
					found := scanDomain(domain, pathList, pathWorkers, results, cfg.SaveContent)
					if found > 0 {
						atomic.AddInt64(&stats.VulnsFound, int64(found))
					}
					atomic.AddInt64(&stats.DomainsScanned, 1)
				}
			}
		}()
	}

	for _, path := range files {
		if ctx.Err() != nil {
			break
		}
		streamChunkFile(ctx, path, scanQueue, stats)
	}

	close(scanQueue)
	scanWg.Wait()
	close(results)
	saveWg.Wait()
}

func streamChunkFile(ctx context.Context, path string, scanQueue chan<- string, stats *Stats) {
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
		atomic.AddInt64(&stats.DomainsPending, 1)
		select {
		case scanQueue <- d:
		case <-ctx.Done():
			return
		}
	}
}
