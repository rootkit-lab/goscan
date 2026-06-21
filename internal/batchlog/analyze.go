package batchlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goscan/internal/checker"
	"goscan/internal/paths"
)

// LoadResults reads results.jsonl from a run directory.
func LoadResults(runDir string) ([]CheckRecord, error) {
	f, err := os.Open(filepath.Join(runDir, "results.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []CheckRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var rec CheckRecord
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}

// ResolveRunDir resolves --last or an explicit path.
func ResolveRunDir(repoRoot, arg string) (string, error) {
	if arg == "" || arg == "--last" {
		latest := filepath.Join(paths.BatchLogsRoot(repoRoot), "latest")
		if target, err := os.Readlink(latest); err == nil {
			if filepath.IsAbs(target) {
				return target, nil
			}
			return filepath.Join(filepath.Dir(latest), target), nil
		}
		return findLatestRun(paths.BatchLogsRoot(repoRoot))
	}
	if strings.HasPrefix(arg, "--last ") {
		n := 1
		fmt.Sscanf(strings.TrimPrefix(arg, "--last "), "%d", &n)
		return findNthLatestRun(paths.BatchLogsRoot(repoRoot), n)
	}
	if !filepath.IsAbs(arg) {
		arg = filepath.Join(repoRoot, arg)
	}
	if st, err := os.Stat(arg); err != nil || !st.IsDir() {
		return "", fmt.Errorf("directório de run inválido: %s", arg)
	}
	return arg, nil
}

func findLatestRun(batchRoot string) (string, error) {
	return findNthLatestRun(batchRoot, 1)
}

func findNthLatestRun(batchRoot string, n int) (string, error) {
	entries, err := os.ReadDir(batchRoot)
	if err != nil {
		return "", fmt.Errorf("sem runs em %s: %w", batchRoot, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "latest" {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("nenhum run encontrado em %s", batchRoot)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	if n < 1 {
		n = 1
	}
	if n > len(dirs) {
		n = len(dirs)
	}
	return filepath.Join(batchRoot, dirs[n-1]), nil
}

var dbEngineScripts = map[string]string{
	"chk-mysql":     "MySQL",
	"chk-postgres":  "PostgreSQL",
	"chk-redis":     "Redis",
	"chk-mongodb":   "MongoDB",
	"chk-memcached": "Memcached",
}

// AnalyzeRun prints a human-readable failure report for a batch run.
func AnalyzeRun(runDir string) error {
	recs, err := LoadResults(runDir)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		fmt.Printf("Run: %s\n", runDir)
		fmt.Println(string(data))
	}
	printRunStats(recs)
	printGroupReport("SMTP", recs, checker.IsSMTPScript)
	for scriptID, title := range dbEngineScripts {
		id := scriptID
		printGroupReport(title, recs, func(s string) bool { return s == id })
	}
	printSuggestions(recs)
	return nil
}

func printRunStats(recs []CheckRecord) {
	ok, fail, skip := 0, 0, 0
	for _, r := range recs {
		switch r.Status {
		case "ok":
			ok++
		case "skip":
			skip++
		default:
			fail++
		}
	}
	fmt.Printf("\nResumo real: OK %d · FAIL %d · SKIP %d · %d checks\n", ok, fail, skip, len(recs))
}

func printGroupReport(title string, recs []CheckRecord, match func(string) bool) {
	byClass := map[string]int{}
	bySnippet := map[string]int{}
	failCount, skipCount, okCount := 0, 0, 0
	for _, r := range recs {
		if !match(r.ScriptID) {
			continue
		}
		switch r.Status {
		case "ok":
			okCount++
		case "skip":
			skipCount++
		case "fail":
			failCount++
			cls := r.ErrorClass
			if cls == "" {
				cls = "other"
			}
			byClass[cls]++
			snip := snippet(r.Summary, 80)
			if snip == "" {
				snip = "(sem summary)"
			}
			bySnippet[snip]++
		}
	}
	if failCount == 0 && skipCount == 0 && okCount == 0 {
		return
	}
	fmt.Printf("\n%s — OK %d · FAIL %d · SKIP %d\n", title, okCount, failCount, skipCount)
	if failCount == 0 {
		return
	}
	fmt.Println("  Por classe:")
	for _, kv := range sortedMap(byClass) {
		fmt.Printf("    %d× %s\n", kv.v, kv.k)
	}
	fmt.Println("  Top mensagens:")
	for i, kv := range sortedMap(bySnippet) {
		if i >= 6 {
			break
		}
		fmt.Printf("    %d× %s\n", kv.v, kv.k)
	}
}

type kv struct{ k string; v int }

func sortedMap(m map[string]int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].v != out[j].v {
			return out[i].v > out[j].v
		}
		return out[i].k < out[j].k
	})
	return out
}

func snippet(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func printSuggestions(recs []CheckRecord) {
	var smtpPolicy, smtpAuth, smtpDNS, smtpClosed int
	var mysqlAuth, mysqlHostDen, mysqlTimeout int
	var pgTimeout, redisDNS int
	var privateSkip int

	for _, r := range recs {
		if r.Status == "skip" && strings.Contains(strings.ToLower(r.Summary), "privad") {
			privateSkip++
		}
		if r.Status != "fail" {
			continue
		}
		switch r.ScriptID {
		case "chk-smtp":
			switch r.ErrorClass {
			case "policy":
				smtpPolicy++
			case "auth":
				smtpAuth++
			case "dns":
				smtpDNS++
			case "closed":
				smtpClosed++
			}
		case "chk-mysql":
			switch r.ErrorClass {
			case "auth":
				mysqlAuth++
			case "host_denied":
				mysqlHostDen++
			case "timeout":
				mysqlTimeout++
			}
		case "chk-postgres":
			if r.ErrorClass == "timeout" {
				pgTimeout++
			}
		case "chk-redis":
			if r.ErrorClass == "dns" {
				redisDNS++
			}
		}
	}

	fmt.Println("\nSugestões:")
	if smtpPolicy > 0 {
		fmt.Println("  → SMTP policy (550/554): credenciais podem estar OK — bloqueio do provider/domínio FROM")
	}
	if smtpAuth > 0 || smtpClosed > 0 {
		fmt.Println("  → SMTP: confirmar MAIL_HOST real (Office365, SendGrid…) e MAIL_ENCRYPTION=tls na 587")
	}
	if smtpDNS > 0 {
		fmt.Println("  → SMTP: evitar smtp.{domain} inventado — usar host do .env ou APP_URL")
	}
	if mysqlAuth > 0 {
		fmt.Println("  → MySQL auth fail: servidor alcançável — credenciais úteis para revisão manual")
	}
	if mysqlHostDen > 0 {
		fmt.Println("  → MySQL host denied (1130): BD responde — IP do scan não autorizado")
	}
	if mysqlTimeout+pgTimeout > 0 {
		fmt.Println("  → DB timeout: IPs privados já SKIP; restantes são hosts públicos offline/firewall")
	}
	if redisDNS > 0 {
		fmt.Println("  → Redis: REDIS_HOST=redis local — usar host do APP_URL, não redis.{domain}")
	}
	if privateSkip > 0 {
		fmt.Printf("  → %d SKIP por IP privado — esperado, batch mais rápido\n", privateSkip)
	}
	if smtpPolicy+smtpAuth+smtpDNS+smtpClosed+mysqlAuth+mysqlHostDen+mysqlTimeout+pgTimeout+redisDNS+privateSkip == 0 {
		fmt.Println("  (nada específico — rever failures/*.txt no run)")
	}
}

// WriteFailureReports writes aggregated failure files under failures/.
func WriteFailureReports(runDir string, recs []CheckRecord) error {
	failDir := filepath.Join(runDir, "failures")
	if err := os.MkdirAll(failDir, 0o755); err != nil {
		return err
	}
	if err := writeGroupFile(filepath.Join(failDir, "smtp-top-errors.txt"), recs, checker.IsSMTPScript); err != nil {
		return err
	}
	for scriptID, title := range dbEngineScripts {
		id := scriptID
		fname := strings.ToLower(title) + "-top-errors.txt"
		if err := writeGroupFile(filepath.Join(failDir, fname), recs, func(s string) bool { return s == id }); err != nil {
			return err
		}
	}
	return writeGroupFile(filepath.Join(failDir, "db-top-errors.txt"), recs, checker.IsDBScript)
}

func writeGroupFile(path string, recs []CheckRecord, match func(string) bool) error {
	bySnippet := map[string]int{}
	for _, r := range recs {
		if !match(r.ScriptID) || r.Status != "fail" {
			continue
		}
		snip := snippet(r.Summary, 120)
		if snip == "" {
			snip = r.ErrorClass
		}
		bySnippet[snip]++
	}
	var b strings.Builder
	for _, kv := range sortedMap(bySnippet) {
		fmt.Fprintf(&b, "%d× %s\n", kv.v, kv.k)
	}
	if b.Len() == 0 {
		b.WriteString("(sem falhas)\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
