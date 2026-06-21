package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"goscan/internal/batchlog"
	"goscan/internal/paths"
	"goscan/internal/scanner"
	"goscan/internal/scripts"
	"goscan/internal/settings"
	"goscan/internal/store"
)

type App struct {
	wails       *application.App
	repoRoot    string // app root (scripts)
	dataRoot    string
	scanDir     string
	dbPath      string
	findingsDir string
	domainStore *store.DomainStore
	findings    *store.FindingsStore
	checkers    *store.CheckerResultsStore
	scriptRun   *scripts.Runner

	scanMu     sync.Mutex
	scanCancel context.CancelFunc

	scriptMu sync.Mutex
	scriptRunActive *activeScriptRun

	batchMu     sync.Mutex
	batchCancel context.CancelFunc
}

type FindingDTO struct {
	ID             int64  `json:"id"`
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	URL            string `json:"url"`
	Confidence     string `json:"confidence"`
	FilePath       string `json:"filePath"`
	ScanRunID      string `json:"scanRunId"`
	FoundAt        string `json:"foundAt"`
	HasCredentials bool   `json:"hasCredentials"`
	IsNew          bool   `json:"isNew"`
}

type FindingsStatsDTO struct {
	Total    int64 `json:"total"`
	Unopened int64 `json:"unopened"`
}

type FindingDetailDTO struct {
	FindingDTO
	Content string `json:"content"`
	AbsPath string `json:"absPath"`
}

type ScriptDTO struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	EnvKeys     []string `json:"envKeys"`
	Interactive bool     `json:"interactive"`
}

type ScanOptsDTO struct {
	Dir         string `json:"dir"`
	Threads     int    `json:"threads"`
	PathWorkers int    `json:"pathWorkers"`
	Fast        bool   `json:"fast"`
	Rescan      bool   `json:"rescan"`
	TimeoutSec  int    `json:"timeoutSec"`
}

type ScanProgressDTO struct {
	DomainsScanned int64 `json:"domainsScanned"`
	VulnsFound     int64 `json:"vulnsFound"`
	DomainsNew     int64 `json:"domainsNew"`
	DomainsPending int64 `json:"domainsPending"`
	Running        bool  `json:"running"`
}

type CheckerResultDTO struct {
	FindingID   int64  `json:"findingId"`
	ScriptID    string `json:"scriptId"`
	ScriptLabel string `json:"scriptLabel"`
	Status      string `json:"status"`
	ExitCode    int    `json:"exitCode"`
	Summary     string `json:"summary"`
	TestedAt    string `json:"testedAt"`
}

type ScriptCheckerStatusDTO struct {
	ScriptID string `json:"scriptId"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Summary  string `json:"summary"`
	TestedAt string `json:"testedAt"`
	ExitCode int    `json:"exitCode"`
	LogPath  string `json:"logPath"`
}

type FindingCheckerOverviewDTO struct {
	FindingID int64                    `json:"findingId"`
	Scripts   []ScriptCheckerStatusDTO `json:"scripts"`
}

type BatchCheckOptsDTO struct {
	FindingID    int64  `json:"findingId"`
	Query        string `json:"query"`
	Confidence   string `json:"confidence"`
	UnopenedOnly bool   `json:"unopenedOnly"`
	ScriptID     string `json:"scriptId"`
	Quick        bool   `json:"quick"`
	Limit        int    `json:"limit"`
	Threads      int    `json:"threads"`
}

type BatchProgressDTO struct {
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
	Running      bool   `json:"running"`
	CheckIndex   int    `json:"checkIndex"`
	CheckTotal   int    `json:"checkTotal"`
	OkCount      int    `json:"okCount"`
	FailCount    int    `json:"failCount"`
	SkipCount    int    `json:"skipCount"`
	Threads      int    `json:"threads"`
	LogPath      string `json:"logPath"`
}

type BatchDoneDTO struct {
	OK     int    `json:"ok"`
	Fail   int    `json:"fail"`
	Skip   int    `json:"skip"`
	Total  int    `json:"total"`
	Secs   int    `json:"secs"`
	LogDir string `json:"logDir"`
}

type SettingsDTO struct {
	Mode               string `json:"mode"`
	DataDir            string `json:"dataDir"`
	ScanDir            string `json:"scanDir"`
	AppRoot            string `json:"appRoot"`
	DefaultProdDataDir string `json:"defaultProdDataDir"`
	PointsToDevRepo    bool   `json:"pointsToDevRepo"`
	NeedsSetup         bool   `json:"needsSetup"`
	Version            string `json:"version"`
}

type activeScriptRun struct {
	findingID int64
	scriptID  string
	output    strings.Builder
}

func NewApp() (*App, error) {
	appRoot, err := paths.AppRoot()
	if err != nil {
		return nil, fmt.Errorf("app root: %w", err)
	}
	a := &App{repoRoot: appRoot}
	if err := a.reloadStores(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) closeStores() {
	if a.domainStore != nil {
		a.domainStore.Close()
		a.domainStore = nil
	}
	a.findings = nil
	a.checkers = nil
	a.scriptRun = nil
}

func (a *App) reloadStores() error {
	dataRoot, err := paths.DataRoot()
	if err != nil {
		return fmt.Errorf("data root: %w", err)
	}
	scanDir, err := paths.ScanInputDir(dataRoot)
	if err != nil {
		return fmt.Errorf("scan dir: %w", err)
	}
	if err := os.MkdirAll(paths.FindingsByDomain(dataRoot), 0755); err != nil {
		return fmt.Errorf("findings dir: %w", err)
	}
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		return fmt.Errorf("scan dir: %w", err)
	}

	dbPath := paths.DefaultDBPath(dataRoot)
	findingsDir := paths.FindingsRoot(dataRoot)

	domainStore, err := store.OpenDomainStore(dbPath)
	if err != nil {
		return fmt.Errorf("db %s: %w", dbPath, err)
	}
	fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
	if err != nil {
		domainStore.Close()
		return fmt.Errorf("findings: %w", err)
	}
	cr, err := store.OpenCheckerResultsStore(domainStore.DB())
	if err != nil {
		domainStore.Close()
		return fmt.Errorf("checker results: %w", err)
	}
	runner, err := scripts.NewRunner(a.repoRoot)
	if err != nil {
		domainStore.Close()
		return fmt.Errorf("scripts/registry: %w", err)
	}

	a.closeStores()
	a.dataRoot = dataRoot
	a.scanDir = scanDir
	a.dbPath = dbPath
	a.findingsDir = findingsDir
	a.domainStore = domainStore
	a.findings = fs
	a.checkers = cr
	a.scriptRun = runner
	return nil
}

func (a *App) GetSettings() SettingsDTO {
	mode := paths.Mode()
	version := paths.InstallVersion(a.repoRoot)
	defProd, _ := paths.DefaultProdDataRoot()
	return SettingsDTO{
		Mode:               mode,
		DataDir:            a.dataRoot,
		ScanDir:            a.scanDir,
		AppRoot:            a.repoRoot,
		DefaultProdDataDir: defProd,
		PointsToDevRepo:    paths.IsDevRepoPath(a.dataRoot) || paths.IsDevRepoPath(a.scanDir),
		NeedsSetup:         paths.ProdNeedsSetup(),
		Version:            version,
	}
}

func (a *App) PickDirectory(title, current string) (string, error) {
	if a.wails == nil {
		return "", fmt.Errorf("interface ainda não pronta")
	}
	start := current
	if start == "" {
		if home, err := os.UserHomeDir(); err == nil {
			start = home
		}
	}
	dlg := a.wails.Dialog.OpenFile().
		SetTitle(title).
		SetDirectory(start).
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true)
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("nenhuma pasta seleccionada")
	}
	return filepath.Clean(path), nil
}

func (a *App) SaveSettings(dataDir, scanDir string) error {
	dataDir = strings.TrimSpace(dataDir)
	scanDir = strings.TrimSpace(scanDir)
	if dataDir == "" {
		return fmt.Errorf("escolha a pasta de dados")
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}
	if paths.Mode() == paths.ModeProd && paths.IsDevRepoPath(absData) {
		def, _ := paths.DefaultProdDataRoot()
		return fmt.Errorf("produção não pode usar o repo de dev — use %s", def)
	}
	var absScan string
	if scanDir != "" {
		absScan, err = filepath.Abs(scanDir)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(absData, 0755); err != nil {
		return fmt.Errorf("criar pasta dados: %w", err)
	}
	if absScan == "" {
		absScan = filepath.Join(absData, "files")
	}
	if err := os.MkdirAll(absScan, 0755); err != nil {
		return fmt.Errorf("criar pasta scan: %w", err)
	}
	if paths.Mode() == paths.ModeProd && paths.IsDevRepoPath(absScan) {
		return fmt.Errorf("pasta de scan não pode ser o repo de dev")
	}
	if err := settings.Save(settings.User{DataDir: absData, ScanDir: absScan}); err != nil {
		return err
	}
	return a.reloadStores()
}

func (a *App) OpenDataDirectory() error {
	return openPathInFileManager(a.dataRoot)
}

func (a *App) OpenScanDirectory() error {
	return openPathInFileManager(a.scanDir)
}

func (a *App) emit(event string, data any) {
	if a.wails != nil {
		a.wails.Event.Emit(event, data)
	}
}

func toFindingDTO(f store.Finding) FindingDTO {
	return FindingDTO{
		ID: f.ID, Domain: f.Domain, Path: f.Path, URL: f.URL,
		Confidence: f.Confidence, FilePath: f.FilePath, ScanRunID: f.ScanRunID,
		FoundAt: f.FoundAt, HasCredentials: f.HasCredentials,
		IsNew: f.OpenedAt == "",
	}
}

func (a *App) FindingsStats() (FindingsStatsDTO, error) {
	s, err := a.findings.Stats()
	if err != nil {
		return FindingsStatsDTO{}, err
	}
	return FindingsStatsDTO{Total: s.Total, Unopened: s.Unopened}, nil
}

func (a *App) SearchFindings(query, confidence string, unopenedOnly bool, limit int) ([]FindingDTO, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := a.findings.Search(store.FindingsFilter{
		Query: query, Confidence: confidence, UnopenedOnly: unopenedOnly, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]FindingDTO, len(items))
	for i, f := range items {
		out[i] = toFindingDTO(f)
	}
	return out, nil
}

func (a *App) GetFinding(id int64) (FindingDetailDTO, error) {
	f, content, err := a.findings.Get(id)
	if err != nil {
		return FindingDetailDTO{}, err
	}
	_ = a.findings.MarkOpened(id)
	f.OpenedAt = time.Now().UTC().Format(time.RFC3339)
	abs := filepath.Join(a.findingsDir, "by-domain", f.FilePath)
	return FindingDetailDTO{
		FindingDTO: toFindingDTO(*f),
		Content:    content,
		AbsPath:    abs,
	}, nil
}

func (a *App) ListScripts() ([]ScriptDTO, error) {
	list := a.scriptRun.List()
	out := make([]ScriptDTO, len(list))
	for i, s := range list {
		out[i] = ScriptDTO{ID: s.ID, Label: s.Label, EnvKeys: s.EnvKeys, Interactive: s.Interactive}
	}
	return out, nil
}

func (a *App) CompatibleScripts(findingID int64) ([]ScriptDTO, error) {
	detail, err := a.GetFinding(findingID)
	if err != nil {
		return nil, err
	}
	list, err := a.scriptRun.CompatibleScripts(detail.AbsPath)
	if err != nil {
		return nil, err
	}
	out := make([]ScriptDTO, len(list))
	for i, s := range list {
		out[i] = ScriptDTO{ID: s.ID, Label: s.Label, EnvKeys: s.EnvKeys, Interactive: s.Interactive}
	}
	return out, nil
}

func (a *App) scriptLabel(scriptID string) string {
	if s, err := a.scriptRun.Find(scriptID); err == nil {
		return s.Label
	}
	return scriptID
}

func toCheckerDTO(findingID int64, r store.CheckerResult, label string) CheckerResultDTO {
	return CheckerResultDTO{
		FindingID: findingID, ScriptID: r.ScriptID, ScriptLabel: label,
		Status: r.Status, ExitCode: r.ExitCode, Summary: r.Summary, TestedAt: r.TestedAt,
	}
}

func (a *App) ListCheckerResults(findingID int64) ([]CheckerResultDTO, error) {
	list, err := a.checkers.ListByFinding(findingID)
	if err != nil {
		return nil, err
	}
	out := make([]CheckerResultDTO, len(list))
	for i, r := range list {
		out[i] = toCheckerDTO(findingID, r, a.scriptLabel(r.ScriptID))
	}
	return out, nil
}

func (a *App) CheckerOverview(findingIDs []int64) ([]FindingCheckerOverviewDTO, error) {
	stored, err := a.checkers.ListByFindings(findingIDs)
	if err != nil {
		return nil, err
	}
	out := make([]FindingCheckerOverviewDTO, 0, len(findingIDs))
	for _, fid := range findingIDs {
		rel, err := a.findings.GetFilePath(fid)
		if err != nil {
			continue
		}
		abs := filepath.Join(a.findingsDir, "by-domain", rel)
		compat, err := a.scriptRun.CompatibleScripts(abs)
		if err != nil {
			continue
		}
		byScript := map[string]store.CheckerResult{}
		for _, r := range stored[fid] {
			if prev, ok := byScript[r.ScriptID]; !ok || r.TestedAt > prev.TestedAt {
				byScript[r.ScriptID] = r
			}
		}
		scripts := make([]ScriptCheckerStatusDTO, 0, len(compat))
		for _, s := range compat {
			st := ScriptCheckerStatusDTO{ScriptID: s.ID, Label: s.Label, Status: "pending"}
			if r, ok := byScript[s.ID]; ok {
				st.Status = r.Status
				st.Summary = r.Summary
				st.TestedAt = r.TestedAt
				st.ExitCode = r.ExitCode
				st.LogPath = r.LogPath
			}
			scripts = append(scripts, st)
		}
		out = append(out, FindingCheckerOverviewDTO{FindingID: fid, Scripts: scripts})
	}
	return out, nil
}

func (a *App) beginScriptRun(findingID int64, scriptID string) {
	a.scriptMu.Lock()
	a.scriptRunActive = &activeScriptRun{findingID: findingID, scriptID: scriptID}
	a.scriptMu.Unlock()
	a.emit("checker:running", map[string]any{"findingId": findingID, "scriptId": scriptID})
}

func (a *App) appendScriptOutput(chunk string) {
	a.scriptMu.Lock()
	defer a.scriptMu.Unlock()
	if a.scriptRunActive == nil {
		return
	}
	if a.scriptRunActive.output.Len() < 12000 {
		a.scriptRunActive.output.WriteString(chunk)
	}
}

func (a *App) finishScriptRun(scriptID string, exitCode int) {
	a.scriptMu.Lock()
	run := a.scriptRunActive
	a.scriptRunActive = nil
	a.scriptMu.Unlock()
	if run == nil || run.scriptID != scriptID {
		return
	}
	if exitCode < 0 {
		return
	}
	output := run.output.String()
	status := classifyCheckerStatus(exitCode, output)
	summary := summarizeOutput(output)
	_ = a.checkers.Save(run.findingID, scriptID, status, exitCode, summary, "", "")
	dto := CheckerResultDTO{
		FindingID: run.findingID, ScriptID: scriptID, ScriptLabel: a.scriptLabel(scriptID),
		Status: status, ExitCode: exitCode, Summary: summary, TestedAt: time.Now().UTC().Format(time.RFC3339),
	}
	a.emit("checker:updated", dto)
}

func (a *App) wrapScriptEmit(findingID int64, scriptID string, base func(string, any)) scripts.EventEmitter {
	a.beginScriptRun(findingID, scriptID)
	return func(event string, data any) {
		switch event {
		case "terminal:data":
			a.appendScriptOutput(fmt.Sprint(data))
		case "terminal:exit", "script:exit":
			code := 0
			if m, ok := data.(map[string]any); ok {
				switch v := m["exitCode"].(type) {
				case int:
					code = v
				case int64:
					code = int(v)
				case float64:
					code = int(v)
				}
			}
			a.finishScriptRun(scriptID, code)
		}
		base(event, data)
	}
}

func (a *App) RunScript(scriptID string, findingID int64) error {
	detail, err := a.GetFinding(findingID)
	if err != nil {
		return err
	}
	s, err := a.scriptRun.Find(scriptID)
	if err != nil {
		return err
	}
	emit := a.wrapScriptEmit(findingID, scriptID, func(event string, data any) { a.emit(event, data) })
	go func() {
		if s.Interactive {
			a.scriptRun.RunInteractive(context.Background(), scriptID, detail.AbsPath, emit)
		} else {
			a.scriptRun.Run(context.Background(), scriptID, detail.AbsPath, emit, 30*time.Second)
		}
	}()
	return nil
}

func (a *App) TerminalInput(data string) error {
	return a.scriptRun.WriteInput(data)
}

func (a *App) TerminalResize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return a.scriptRun.ResizeTerminal(uint16(cols), uint16(rows))
}

func (a *App) CancelScript() {
	a.scriptRun.Cancel()
}

func (a *App) saveBatchResult(findingID int64, scriptID, status string, exitCode int, summary, runID, logPath string) {
	_ = a.checkers.Save(findingID, scriptID, status, exitCode, summary, runID, logPath)
	a.emit("checker:updated", CheckerResultDTO{
		FindingID: findingID, ScriptID: scriptID, ScriptLabel: a.scriptLabel(scriptID),
		Status: status, ExitCode: exitCode, Summary: summary, TestedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *App) StartBatchCheck(opts BatchCheckOptsDTO) error {
	a.batchMu.Lock()
	if a.batchCancel != nil {
		a.batchMu.Unlock()
		return fmt.Errorf("batch já em execução")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.batchCancel = cancel
	a.batchMu.Unlock()

	planOpts := scripts.BatchPlanOpts{ScriptID: opts.ScriptID, Quick: opts.Quick}

	go func() {
		defer func() {
			a.batchMu.Lock()
			a.batchCancel = nil
			a.batchMu.Unlock()
		}()

		var findings []store.Finding
		var err error
		if opts.FindingID > 0 {
			f, _, e := a.findings.Get(opts.FindingID)
			if e != nil {
				a.emit("batch:output", fmt.Sprintf("Erro: %v", e))
				a.emit("batch:done", BatchDoneDTO{})
				return
			}
			findings = []store.Finding{*f}
		} else {
			limit := opts.Limit
			if limit <= 0 {
				limit = 500
			}
			findings, err = a.findings.Search(store.FindingsFilter{
				Query: opts.Query, Confidence: opts.Confidence, UnopenedOnly: opts.UnopenedOnly, Limit: limit,
			})
			if err != nil {
				a.emit("batch:output", fmt.Sprintf("Erro: %v", err))
				a.emit("batch:done", BatchDoneDTO{})
				return
			}
		}

		var items []scripts.BatchItem
		for _, f := range findings {
			if ctx.Err() != nil {
				break
			}
			abs := filepath.Join(a.findingsDir, "by-domain", f.FilePath)
			part, e := a.scriptRun.PlanBatch(abs, f.Domain, f.ID, planOpts)
			if e != nil {
				continue
			}
			items = append(items, part...)
		}

		if len(items) == 0 {
			a.emit("batch:output", "Nenhum checker compatível.")
			a.emit("batch:done", BatchDoneDTO{})
			return
		}

		a.emit("batch:output", fmt.Sprintf("Batch start — %d findings · %d checks · %d threads", len(findings), len(items), batchThreads(opts.Threads)))
		a.emit("batch:progress", BatchProgressDTO{
			Running: true, Line: "A iniciar…", CheckTotal: len(items), Threads: batchThreads(opts.Threads),
		})
		start := time.Now()
		runID := paths.NewBatchRunID()
		var logWriter *batchlog.Writer
		if w, err := batchlog.NewWriter(batchlog.StartOpts{
			RepoRoot: a.dataRoot,
			RunID:    runID,
			Threads:  batchThreads(opts.Threads),
			Findings: len(findings),
			Checks:   len(items),
			ManifestOpts: batchlog.ManifestOpts{
				Quick: opts.Quick, Limit: opts.Limit, ScriptID: opts.ScriptID,
				FindingID: opts.FindingID, UnopenedOnly: opts.UnopenedOnly,
			},
		}); err != nil {
			a.emit("batch:output", fmt.Sprintf("⚠ logs: %v", err))
		} else {
			logWriter = w
			a.emit("batch:output", fmt.Sprintf("Logs → %s", w.RunDir()))
		}

		workers := batchThreads(opts.Threads)
		stats := a.scriptRun.ExecuteBatch(ctx, items, scripts.BatchExecOpts{
			Workers:   workers,
			LogWriter: logWriter,
			OnProgress: func(p scripts.BatchProgress) {
				a.saveBatchResult(p.FindingID, p.ScriptID, p.Status, p.ExitCode, p.Summary, runID, p.LogPath)
				a.emit("batch:output", p.Line)
				if logWriter != nil {
					logWriter.AppendSummaryLine(p.Line)
				}
				a.emit("batch:progress", BatchProgressDTO{
					FindingIndex: p.FindingIndex, FindingTotal: p.FindingTotal,
					FindingID: p.FindingID, Domain: p.Domain,
					ScriptIndex: p.ScriptIndex, ScriptTotal: p.ScriptTotal,
					ScriptID: p.ScriptID, ScriptLabel: p.ScriptLabel,
					Status: p.Status, Summary: p.Summary, ExitCode: p.ExitCode,
					Line: p.Line, Running: true,
					CheckIndex: p.CheckIndex, CheckTotal: p.CheckTotal,
					OkCount: p.OkCount, FailCount: p.FailCount, SkipCount: p.SkipCount,
					Threads: p.Threads, LogPath: p.LogPath,
				})
			},
		})

		elapsed := time.Since(start)
		doneLine := scripts.FormatBatchDone(stats, elapsed)
		logDir := ""
		if logWriter != nil {
			logWriter.AppendSummaryLine(doneLine)
			_ = logWriter.Finish(batchlog.FinishStats{
				OK: stats.OK, Fail: stats.Fail, Skip: stats.Skip,
			}, elapsed)
			logDir = logWriter.RunDir()
			_ = a.checkers.SaveRun(store.CheckerRun{
				RunID: runID, StartedAt: start.UTC().Format(time.RFC3339),
				OK: stats.OK, Fail: stats.Fail, Skip: stats.Skip, LogDir: logDir,
			})
			_ = logWriter.Close()
		}
		a.emit("batch:output", doneLine)
		a.emit("batch:done", BatchDoneDTO{
			OK: stats.OK, Fail: stats.Fail, Skip: stats.Skip, Total: stats.Total,
			Secs: int(elapsed.Seconds()), LogDir: logDir,
		})
		a.emit("batch:progress", BatchProgressDTO{Running: false})
	}()
	return nil
}

func batchThreads(n int) int {
	if n <= 1 {
		return 1
	}
	if n > 16 {
		return 16
	}
	return n
}

func (a *App) CancelBatchCheck() {
	a.batchMu.Lock()
	defer a.batchMu.Unlock()
	if a.batchCancel != nil {
		a.batchCancel()
		a.batchCancel = nil
	}
	a.scriptRun.Cancel()
}

func (a *App) OpenBatchLogDir(dir string) error {
	if dir == "" {
		latest := filepath.Join(paths.BatchLogsRoot(a.dataRoot), "latest")
		if target, err := os.Readlink(latest); err == nil {
			if !filepath.IsAbs(target) {
				dir = filepath.Join(filepath.Dir(latest), target)
			} else {
				dir = target
			}
		} else if runDir, err := batchlog.ResolveRunDir(a.dataRoot, "--last"); err == nil {
			dir = runDir
		} else {
			return fmt.Errorf("nenhum log de batch encontrado")
		}
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("directório inválido: %s", dir)
	}
	return openPathInFileManager(dir)
}

func openPathInFileManager(path string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("explorer", path).Start()
	default:
		return fmt.Errorf("abrir pasta não suportado em %s", runtime.GOOS)
	}
}

func (a *App) StartScan(opts ScanOptsDTO) error {
	a.scanMu.Lock()
	if a.scanCancel != nil {
		a.scanMu.Unlock()
		return fmt.Errorf("scan já em execução")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.scanMu.Unlock()

	dir := opts.Dir
	if dir == "" {
		dir = a.scanDir
	}
	threads := opts.Threads
	if threads <= 0 {
		threads = 50
	}
	pathWorkers := opts.PathWorkers
	if pathWorkers <= 0 {
		pathWorkers = 8
	}
	timeout := opts.TimeoutSec
	if timeout <= 0 {
		timeout = 8
	}

	cfg := &scanner.Config{
		RepoRoot:    a.dataRoot,
		Dir:         dir,
		DBPath:      a.dbPath,
		FindingsDir: a.findingsDir,
		Threads:     threads,
		PathWorkers: pathWorkers,
		Fast:        opts.Fast,
		Rescan:      opts.Rescan,
		ScanVulns:   true,
		SaveContent: true,
		Timeout:     time.Duration(timeout) * time.Second,
		OnProgress: func(s scanner.Stats) {
			a.emit("scan:progress", ScanProgressDTO{
				DomainsScanned: s.DomainsScanned,
				VulnsFound:     s.VulnsFound,
				DomainsNew:     s.DomainsNew,
				DomainsPending: s.DomainsPending,
				Running:        true,
			})
		},
		OnFound: func(r scanner.VulnResult) {
			a.emit("scan:found", map[string]string{
				"domain": r.Domain,
				"url":    r.URL,
				"path":   r.Path,
			})
			a.emit("scan:output", fmt.Sprintf("FOUND %s %s (%s)", r.Domain, r.Path, r.URL))
		},
	}

	go func() {
		defer func() {
			a.scanMu.Lock()
			a.scanCancel = nil
			a.scanMu.Unlock()
			a.emit("scan:progress", ScanProgressDTO{Running: false})
			a.emit("scan:output", "Scan concluído.")
		}()
		a.emit("scan:output", fmt.Sprintf("Iniciando scan em %s…", dir))
		_ = scanner.Run(ctx, cfg)
	}()
	return nil
}

func (a *App) CancelScan() {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	if a.scanCancel != nil {
		a.scanCancel()
		a.scanCancel = nil
	}
}

func (a *App) Shutdown() {
	a.CancelBatchCheck()
	a.CancelScan()
	if a.domainStore != nil {
		a.domainStore.Close()
	}
}
