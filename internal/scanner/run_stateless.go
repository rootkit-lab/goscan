//go:build nosqlite

package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goscan/internal/remotefindings"
)

type fileFindings struct{ fs *remotefindings.Store }

func (f *fileFindings) SaveFinding(domain, path, url, confidence, scanRunID string, content []byte, hasCredentials bool) (string, error) {
	return f.fs.SaveFinding(domain, path, url, confidence, scanRunID, content, hasCredentials)
}

func (f *fileFindings) DomainFileName(domain, path string) string {
	return remotefindings.DomainFileName(domain, path)
}

// RunStateless scans a batch from text files without SQLite (remote workers).
func RunStateless(ctx context.Context, cfg *Config) error {
	log.SetOutput(io.Discard)
	initHTTPClient(cfg.Timeout)

	if cfg.RunID == "" {
		cfg.RunID = time.Now().Format("20060102_150405")
	}
	runID = cfg.RunID

	findingsDir := cfg.FindingsDir
	if findingsDir == "" {
		findingsDir = "var/findings"
	}

	fs, err := remotefindings.Open(findingsDir)
	if err != nil {
		return fmt.Errorf("findings: %w", err)
	}
	activeFindings = &fileFindings{fs: fs}
	currentCfg = cfg

	files := findInputFiles(cfg.Dir)
	if len(files) == 0 {
		return fmt.Errorf("nenhum ficheiro de entrada (.txt ou .env)")
	}

	stats := &Stats{StartTime: time.Now()}
	batchTotal := int64(cfg.BatchSize)
	progressInterval := 2 * time.Second
	if cfg.HubProgress != nil {
		progressInterval = 500 * time.Millisecond
	}
	if cfg.OnProgress != nil || cfg.ProgressJSON || cfg.HubProgress != nil {
		if cfg.ProgressJSON && cfg.HubProgress == nil && batchTotal > 0 {
			fmt.Fprintf(os.Stderr, "@goscan/progress scanned=0 vulns=0 total=%d\n", batchTotal)
		}
		go func() {
			ticker := time.NewTicker(progressInterval)
			defer ticker.Stop()
			var lastScanned int64
			var lastTick time.Time
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					scanned := atomic.LoadInt64(&stats.DomainsScanned)
					vulns := atomic.LoadInt64(&stats.VulnsFound)
					if batchTotal <= 0 {
						batchTotal = atomic.LoadInt64(&stats.DomainsPending)
					}
					if cfg.ProgressJSON && cfg.HubProgress == nil {
						fmt.Fprintf(os.Stderr, "@goscan/progress scanned=%d vulns=%d total=%d\n", scanned, vulns, batchTotal)
					}
					if cfg.OnProgress != nil {
						cfg.OnProgress(*stats)
					}
					if cfg.HubProgress != nil {
						var rate int64
						if !lastTick.IsZero() && scanned > lastScanned {
							sec := time.Since(lastTick).Seconds()
							if sec > 0 {
								rate = int64(float64(scanned-lastScanned) / sec)
							}
						}
						cfg.HubProgress(scanned, vulns, batchTotal, rate)
						lastScanned = scanned
						lastTick = time.Now()
					}
				}
			}
		}()
	}

	runStatelessScan(ctx, files, cfg, stats)
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func runStatelessScan(ctx context.Context, files []string, cfg *Config, stats *Stats) {
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

	go reportPipelineProgress(ctx, stats)

	for _, path := range files {
		if ctx.Err() != nil {
			break
		}
		if isDomainListFile(path) {
			streamDomainList(ctx, path, scanQueue, stats)
		} else {
			streamEnvDomains(ctx, path, scanQueue, stats)
		}
		atomic.AddInt64(&stats.FilesProcessed, 1)
	}

	close(scanQueue)
	scanWg.Wait()
	close(results)
	saveWg.Wait()
}

func streamDomainList(ctx context.Context, path string, scanQueue chan<- string, stats *Stats) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	seen := make(map[string]struct{})
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
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		atomic.AddInt64(&stats.DomainsPending, 1)
		select {
		case scanQueue <- d:
		case <-ctx.Done():
			return
		}
	}
}

func streamEnvDomains(ctx context.Context, path string, scanQueue chan<- string, stats *Stats) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	seen := make(map[string]struct{})

	add := func(d string) {
		if ctx.Err() != nil {
			return
		}
		if !isValidDomain(d) {
			return
		}
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		atomic.AddInt64(&stats.DomainsPending, 1)
		select {
		case scanQueue <- d:
		case <-ctx.Done():
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
