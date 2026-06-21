package main

import (
	"context"
	"errors"
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
	"goscan/internal/remoteworker"
	"goscan/internal/scanorch"
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
	OpenedAt       string `json:"openedAt"`
	ModifiedAt     string `json:"modifiedAt"`
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
	Dir          string   `json:"dir"`
	Threads      int      `json:"threads"`
	PathWorkers  int      `json:"pathWorkers"`
	Fast         bool     `json:"fast"`
	Rescan       bool     `json:"rescan"`
	TimeoutSec   int      `json:"timeoutSec"`
	Targets      []string `json:"targets"`
	DeployRemote bool     `json:"deployRemote"`
}

type ScanWorkerProgressDTO struct {
	WorkerID       string `json:"workerId"`
	WorkerName     string `json:"workerName"`
	DomainsScanned int64  `json:"domainsScanned"`
	VulnsFound     int64  `json:"vulnsFound"`
	DomainsTotal   int64  `json:"domainsTotal"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	Running        bool   `json:"running"`
	PhasePercent   int    `json:"phasePercent,omitempty"`
	PhaseLabel     string `json:"phaseLabel,omitempty"`
}

type ScanProgressDTO struct {
	DomainsScanned int64 `json:"domainsScanned"` // sessão actual (ondas concluídas + onda em curso)
	VulnsFound     int64 `json:"vulnsFound"`
	DomainsNew     int64 `json:"domainsNew"`
	DomainsPending int64 `json:"domainsPending"` // fila central
	Wave           int   `json:"wave,omitempty"`
	WaveBatchSize  int   `json:"waveBatchSize,omitempty"`
	WaveScanned    int64 `json:"waveScanned,omitempty"`
	SessionScanned int64 `json:"sessionScanned,omitempty"`
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
	FindingID     int64  `json:"findingId"`
	Query         string `json:"query"`
	Confidence    string `json:"confidence"`
	UnopenedOnly  bool   `json:"unopenedOnly"`
	ScriptID      string `json:"scriptId"`
	Quick         bool   `json:"quick"`
	UntestedOnly  bool   `json:"untestedOnly"`
	ForceRecheck  bool   `json:"forceRecheck"`
	Limit         int    `json:"limit"`
	Threads       int    `json:"threads"`
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
	Mode                string            `json:"mode"`
	DataDir             string            `json:"dataDir"`
	ScanDir             string            `json:"scanDir"`
	AppRoot             string            `json:"appRoot"`
	DefaultProdDataDir  string            `json:"defaultProdDataDir"`
	PointsToDevRepo      bool              `json:"pointsToDevRepo"`
	NeedsSetup          bool              `json:"needsSetup"`
	Version             string            `json:"version"`
	PythonPath          string            `json:"pythonPath"`
	PythonPathEffective string            `json:"pythonPathEffective"`
	NotifyEnvFound      bool              `json:"notifyEnvFound"`
	NotifyScriptOk      bool              `json:"notifyScriptOk"`
	SoundEnvFound       bool              `json:"soundEnvFound"`
	SoundScriptOk       bool              `json:"soundScriptOk"`
	Workers             []RemoteWorkerDTO `json:"workers"`
	DeployRepoURL       string            `json:"deployRepoUrl"`
	DeployRepoRef       string            `json:"deployRepoRef"`
	DeployRepoMethod    string            `json:"deployRepoMethod"`
	DeployRepoHasToken  bool              `json:"deployRepoHasToken"`
	HubEnabled          bool              `json:"hubEnabled"`
}

type RemoteWorkerDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	AuthType      string `json:"authType"`
	KeyPath       string `json:"keyPath"`
	ExecMode      string `json:"execMode"`
	APIPort       int    `json:"apiPort"`
	Enabled       bool   `json:"enabled"`
	HasPassword   bool   `json:"hasPassword"`
	RemoteVersion string `json:"remoteVersion,omitempty"`
}

type RemoteWorkerTestResultDTO struct {
	OK             bool   `json:"ok"`
	RemoteVersion  string `json:"remoteVersion"`
	Error          string `json:"error,omitempty"`
}

type SettingsSaveDTO struct {
	DataDir        string            `json:"dataDir"`
	ScanDir        string            `json:"scanDir"`
	PythonPath     string            `json:"pythonPath"`
	NotifyEnvFound bool              `json:"notifyEnvFound"`
	NotifyScriptOk bool              `json:"notifyScriptOk"`
	SoundEnvFound  bool              `json:"soundEnvFound"`
	SoundScriptOk  bool              `json:"soundScriptOk"`
	Workers        []RemoteWorkerSaveDTO `json:"workers"`
	DeployRepoURL    string            `json:"deployRepoUrl"`
	DeployRepoRef    string            `json:"deployRepoRef"`
	DeployRepoToken  string            `json:"deployRepoToken"`
	DeployRepoMethod string            `json:"deployRepoMethod"`
	HubEnabled     bool              `json:"hubEnabled"`
}

type RemoteWorkerSaveDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	AuthType      string `json:"authType"`
	Password      string `json:"password"`
	KeyPath       string `json:"keyPath"`
	KeyPassphrase string `json:"keyPassphrase"`
	ExecMode      string `json:"execMode"`
	APIPort       int    `json:"apiPort"`
	APIToken      string `json:"apiToken"`
	Enabled       bool   `json:"enabled"`
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
	user, _ := settings.Load()
	notifyEnv := user.NotifyEnvFoundOrDefault()
	notifyOk := user.NotifyScriptOkOrDefault()
	soundEnv := user.SoundEnvFoundOrDefault()
	soundOk := user.SoundScriptOkOrDefault()
	hubEnabled := user.HubEnabledOrDefault()
	return SettingsDTO{
		Mode:                mode,
		DataDir:             a.dataRoot,
		ScanDir:             a.scanDir,
		AppRoot:             a.repoRoot,
		DefaultProdDataDir:  defProd,
		PointsToDevRepo:     paths.IsDevRepoPath(a.dataRoot) || paths.IsDevRepoPath(a.scanDir),
		NeedsSetup:          paths.ProdNeedsSetup(),
		Version:             version,
		PythonPath:          user.PythonPath,
		PythonPathEffective: scripts.PythonExecutable(a.repoRoot),
		NotifyEnvFound:      notifyEnv,
		NotifyScriptOk:      notifyOk,
		SoundEnvFound:       soundEnv,
		SoundScriptOk:       soundOk,
		Workers:             workersToDTO(user.Workers),
		DeployRepoURL:       user.DeployRepo.URL,
		DeployRepoRef:       user.DeployRepo.Ref,
		DeployRepoMethod:    user.DeployRepo.Method,
		DeployRepoHasToken:  user.DeployRepo.Token != "",
		HubEnabled:          hubEnabled,
	}
}

func workersToDTO(list []settings.RemoteWorker) []RemoteWorkerDTO {
	out := make([]RemoteWorkerDTO, 0, len(list))
	for _, w := range list {
		w = w.Normalized()
		out = append(out, RemoteWorkerDTO{
			ID: w.ID, Name: w.Name, Host: w.Host, Port: w.Port, User: w.User,
			AuthType: w.AuthType, KeyPath: w.KeyPath, ExecMode: w.ExecMode,
			APIPort: w.APIPort, Enabled: w.Enabled, HasPassword: w.Password != "",
		})
	}
	return out
}

func workerFromSaveDTO(d RemoteWorkerSaveDTO) settings.RemoteWorker {
	return settings.RemoteWorker{
		ID: d.ID, Name: d.Name, Host: d.Host, Port: d.Port, User: d.User,
		AuthType: d.AuthType, Password: d.Password, KeyPath: d.KeyPath,
		KeyPassphrase: d.KeyPassphrase, ExecMode: d.ExecMode, APIPort: d.APIPort,
		APIToken: d.APIToken, Enabled: d.Enabled,
	}.Normalized()
}

func (a *App) PickKeyFile(title, current string) (string, error) {
	if a.wails == nil {
		return "", fmt.Errorf("interface ainda não pronta")
	}
	start := current
	if start == "" {
		if home, err := os.UserHomeDir(); err == nil {
			start = filepath.Join(home, ".ssh")
		}
	} else {
		start = filepath.Dir(start)
	}
	dlg := a.wails.Dialog.OpenFile().
		SetTitle(title).
		SetDirectory(start).
		CanChooseFiles(true).
		CanChooseDirectories(false)
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("nenhum ficheiro seleccionado")
	}
	return filepath.Clean(path), nil
}

func (a *App) TestRemoteWorker(w RemoteWorkerSaveDTO) RemoteWorkerTestResultDTO {
	worker := workerFromSaveDTO(w)
	if worker.Password == "" && w.ID != "" {
		if user, err := settings.Load(); err == nil {
			if prev, ok := user.WorkerByID(w.ID); ok && prev.Password != "" {
				worker.Password = prev.Password
			}
		}
	}
	cfg := remoteworker.ConfigFrom(worker, a.repoRoot, paths.InstallVersion(a.repoRoot), userDeployRepo())
	ver, err := remoteworker.TestConnection(cfg)
	if err != nil {
		return RemoteWorkerTestResultDTO{OK: false, Error: err.Error()}
	}
	if ver == "none" {
		ver = ""
	}
	return RemoteWorkerTestResultDTO{OK: true, RemoteVersion: ver}
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

func (a *App) PickPythonExecutable(current string) (string, error) {
	if a.wails == nil {
		return "", fmt.Errorf("interface ainda não pronta")
	}
	start := current
	if start == "" {
		start = filepath.Join(a.repoRoot, "scripts", ".venv", "bin")
		if st, err := os.Stat(start); err != nil || !st.IsDir() {
			if home, err := os.UserHomeDir(); err == nil {
				start = home
			}
		}
	} else {
		start = filepath.Dir(start)
	}
	dlg := a.wails.Dialog.OpenFile().
		SetTitle("Executável Python").
		SetDirectory(start).
		CanChooseFiles(true).
		CanChooseDirectories(false)
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("nenhum ficheiro seleccionado")
	}
	path = filepath.Clean(path)
	if err := validatePythonExecutable(path); err != nil {
		return "", err
	}
	return path, nil
}

func validatePythonExecutable(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("python: %w", err)
	}
	if st.IsDir() {
		return fmt.Errorf("python: é uma pasta")
	}
	if runtime.GOOS != "windows" && st.Mode()&0111 == 0 {
		return fmt.Errorf("python: ficheiro não executável")
	}
	return nil
}

func (a *App) SaveSettings(opts SettingsSaveDTO) error {
	dataDir := strings.TrimSpace(opts.DataDir)
	scanDir := strings.TrimSpace(opts.ScanDir)
	pythonPath := strings.TrimSpace(opts.PythonPath)
	if dataDir == "" {
		return fmt.Errorf("escolha a pasta de dados")
	}
	if pythonPath != "" {
		if err := validatePythonExecutable(pythonPath); err != nil {
			return err
		}
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
	ne, ns, se, ss := opts.NotifyEnvFound, opts.NotifyScriptOk, opts.SoundEnvFound, opts.SoundScriptOk
	existing, _ := settings.Load()
	workers := settings.MergeWorkers(existing.Workers, workersFromSaveDTO(opts.Workers))
	deployRepo := settings.MergeDeployRepo(existing.DeployRepo, settings.DeployRepo{
		URL: opts.DeployRepoURL, Ref: opts.DeployRepoRef,
		Token: opts.DeployRepoToken, Method: opts.DeployRepoMethod,
	})
	hubEnabled := opts.HubEnabled
	if err := settings.Save(settings.User{
		DataDir: absData, ScanDir: absScan, PythonPath: pythonPath,
		NotifyEnvFound: &ne, NotifyScriptOk: &ns, SoundEnvFound: &se, SoundScriptOk: &ss,
		Workers: workers, DeployRepo: deployRepo, HubEnabled: &hubEnabled,
	}); err != nil {
		return err
	}
	return a.reloadStores()
}

func workersFromSaveDTO(list []RemoteWorkerSaveDTO) []settings.RemoteWorker {
	out := make([]settings.RemoteWorker, 0, len(list))
	for _, d := range list {
		out = append(out, workerFromSaveDTO(d))
	}
	return out
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

func (a *App) emitForFinding(findingID int64, event string, data any) {
	if a.wails == nil {
		return
	}
	name := fmt.Sprintf("editor-%d", findingID)
	if w, ok := a.wails.Window.GetByName(name); ok {
		w.EmitEvent(event, data)
		return
	}
	a.emit(event, data)
}

func toFindingDTO(f store.Finding) FindingDTO {
	return FindingDTO{
		ID: f.ID, Domain: f.Domain, Path: f.Path, URL: f.URL,
		Confidence: f.Confidence, FilePath: f.FilePath, ScanRunID: f.ScanRunID,
		FoundAt: f.FoundAt, OpenedAt: f.OpenedAt, HasCredentials: f.HasCredentials,
		IsNew: f.OpenedAt == "",
	}
}

func (a *App) enrichFindingDTO(f store.Finding) FindingDTO {
	dto := toFindingDTO(f)
	abs := filepath.Join(a.findingsDir, "by-domain", f.FilePath)
	if st, err := os.Stat(abs); err == nil {
		dto.ModifiedAt = st.ModTime().UTC().Format(time.RFC3339)
	}
	return dto
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
		out[i] = a.enrichFindingDTO(f)
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

func (a *App) OpenEditorWindow(findingID int64) error {
	if a.wails == nil {
		return fmt.Errorf("app not ready")
	}
	name := fmt.Sprintf("editor-%d", findingID)
	if existing, ok := a.wails.Window.GetByName(name); ok {
		existing.Show()
		existing.Focus()
		return nil
	}
	f, _, err := a.findings.Get(findingID)
	if err != nil {
		return err
	}
	_ = a.findings.MarkOpened(findingID)
	title := fmt.Sprintf("%s%s — goscan", f.Domain, f.Path)
	a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      name,
		Title:     title,
		Width:     1100,
		Height:    800,
		MinWidth:  800,
		MinHeight: 500,
		URL:       fmt.Sprintf("/?window=editor&findingId=%d", findingID),
	})
	return nil
}

func (a *App) FocusMainWindow() error {
	if a.wails == nil {
		return fmt.Errorf("app not ready")
	}
	if w, ok := a.wails.Window.GetByName("main"); ok {
		w.Show()
		w.Focus()
	}
	return nil
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
	a.emitForFinding(findingID, "checker:running", map[string]any{"findingId": findingID, "scriptId": scriptID})
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
	a.emitForFinding(run.findingID, "checker:updated", dto)
	a.emit("checker:updated", dto)
}

func enrichWithFindingID(data any, findingID int64) any {
	switch v := data.(type) {
	case map[string]string:
		m := make(map[string]any, len(v)+1)
		for k, val := range v {
			m[k] = val
		}
		m["findingId"] = findingID
		return m
	case map[string]any:
		m := make(map[string]any, len(v)+1)
		for k, val := range v {
			m[k] = val
		}
		m["findingId"] = findingID
		return m
	case string:
		return map[string]any{"findingId": findingID, "chunk": v}
	default:
		return map[string]any{"findingId": findingID, "payload": data}
	}
}

func isScriptStreamEvent(event string) bool {
	switch event {
	case "terminal:start", "terminal:data", "terminal:exit", "script:stdout", "script:stderr", "script:exit":
		return true
	default:
		return false
	}
}

func (a *App) wrapScriptEmit(findingID int64, scriptID string, base func(string, any)) scripts.EventEmitter {
	a.beginScriptRun(findingID, scriptID)
	return func(event string, data any) {
		switch event {
		case "terminal:data", "script:stdout", "script:stderr":
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
		if isScriptStreamEvent(event) {
			base(event, enrichWithFindingID(data, findingID))
		} else {
			base(event, data)
		}
	}
}

func (a *App) RunScript(scriptID string, findingID int64) error {
	detail, err := a.GetFinding(findingID)
	if err != nil {
		return err
	}
	if _, err := a.scriptRun.Find(scriptID); err != nil {
		return err
	}
	emit := a.wrapScriptEmit(findingID, scriptID, func(event string, data any) { a.emitForFinding(findingID, event, data) })
	go func() {
		a.scriptRun.RunBatch(context.Background(), scriptID, detail.AbsPath, emit)
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

		if opts.UntestedOnly && !opts.ForceRecheck {
			findingIDs := batchFindingIDs(items)
			tested, e := a.checkers.ListByFindings(findingIDs)
			if e != nil {
				a.emit("batch:output", fmt.Sprintf("Erro: %v", e))
				a.emit("batch:done", BatchDoneDTO{})
				return
			}
			before := len(items)
			items = filterUntestedBatchItems(items, tested)
			if skipped := before - len(items); skipped > 0 {
				a.emit("batch:output", fmt.Sprintf("Ignorados %d checks já testados", skipped))
			}
		}

		if len(items) == 0 {
			a.emit("batch:output", "Nenhum checker por testar (todos já testados).")
			a.emit("batch:done", BatchDoneDTO{})
			return
		}

		modeLabel := ""
		if opts.ForceRecheck {
			modeLabel = " · recheck forçado"
		} else if opts.UntestedOnly {
			modeLabel = " · só por testar"
		}
		a.emit("batch:output", fmt.Sprintf("Batch start — %d findings · %d checks · %d threads%s", len(findings), len(items), batchThreads(opts.Threads), modeLabel))
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
				UntestedOnly: opts.UntestedOnly, ForceRecheck: opts.ForceRecheck,
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

func batchFindingIDs(items []scripts.BatchItem) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	for _, it := range items {
		if !seen[it.FindingID] {
			seen[it.FindingID] = true
			ids = append(ids, it.FindingID)
		}
	}
	return ids
}

func filterUntestedBatchItems(items []scripts.BatchItem, tested map[int64][]store.CheckerResult) []scripts.BatchItem {
	pairs := map[string]bool{}
	for fid, results := range tested {
		for _, r := range results {
			pairs[fmt.Sprintf("%d:%s", fid, r.ScriptID)] = true
		}
	}
	out := make([]scripts.BatchItem, 0, len(items))
	for _, it := range items {
		key := fmt.Sprintf("%d:%s", it.FindingID, it.Script.ID)
		if !pairs[key] {
			out = append(out, it)
		}
	}
	return out
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

	includeLocal, remoteWorkers, err := a.resolveScanTargets(opts.Targets)
	if err != nil {
		a.scanMu.Lock()
		a.scanCancel = nil
		a.scanMu.Unlock()
		return err
	}

	if len(remoteWorkers) > 0 {
		a.emit("scan:progress", ScanProgressDTO{Running: true})
		go a.runOrchestratedScan(ctx, dir, includeLocal, remoteWorkers, opts, threads, pathWorkers, timeout)
		return nil
	}

	a.emit("scan:progress", ScanProgressDTO{Running: true})
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
				"domain":   r.Domain,
				"url":      r.URL,
				"path":     r.Path,
				"workerId": "local",
			})
			a.emit("scan:output", fmt.Sprintf("FOUND %s %s (%s)", r.Domain, r.Path, r.URL))
		},
	}

	go func() {
		defer a.finishScan()
		a.emit("scan:output", fmt.Sprintf("Iniciando scan em %s…", dir))
		_ = scanner.Run(ctx, cfg)
		a.emit("scan:output", "Scan concluído.")
	}()
	return nil
}

func (a *App) resolveScanTargets(targets []string) (includeLocal bool, workers []settings.RemoteWorker, err error) {
	user, _ := settings.Load()
	if len(targets) == 0 {
		return true, nil, nil
	}
	seen := map[string]bool{}
	for _, t := range targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if t == "local" {
			includeLocal = true
			continue
		}
		w, ok := user.WorkerByID(t)
		if !ok {
			return false, nil, fmt.Errorf("worker desconhecido: %s", t)
		}
		if !w.Enabled {
			return false, nil, fmt.Errorf("worker desactivado: %s", w.Name)
		}
		if seen[w.ID] {
			continue
		}
		seen[w.ID] = true
		workers = append(workers, w)
	}
	if !includeLocal && len(workers) == 0 {
		return false, nil, fmt.Errorf("seleccione pelo menos um destino")
	}
	return includeLocal, workers, nil
}

func (a *App) runOrchestratedScan(ctx context.Context, dir string, includeLocal bool, remoteWorkers []settings.RemoteWorker, opts ScanOptsDTO, threads, pathWorkers, timeout int) {
	defer a.finishScan()
	runID := paths.NewBatchRunID()
	a.emit("scan:output", fmt.Sprintf("Scan orquestrado %s…", runID))
	user, _ := settings.Load()

	// Mostrar workers imediatamente como "a preparar"
	if includeLocal {
		a.emit("scan:worker-progress", ScanWorkerProgressDTO{
			WorkerID: "local", WorkerName: "Local", Status: "preparing", Running: true,
		})
	}
	for _, w := range remoteWorkers {
		a.emit("scan:worker-progress", ScanWorkerProgressDTO{
			WorkerID: w.ID, WorkerName: w.Name, Status: "preparing", Running: true,
		})
	}
	workerTotals := map[string]scanorch.WorkerProgress{}
	var progressMu sync.Mutex
	var sessionScanned int64
	var sessionVulns int64
	var chunkBatchSize int
	emitAggregate := func() {
		progressMu.Lock()
		defer progressMu.Unlock()
		var chunkScanned, vulns int64
		active := 0
		for _, p := range workerTotals {
			chunkScanned += p.DomainsScanned
			vulns += p.VulnsFound
			if p.Running {
				active++
			}
		}
		if sessionVulns > vulns {
			vulns = sessionVulns
		}
		centralPending := a.domainStore.CountPending(opts.Rescan)
		totalSession := sessionScanned + chunkScanned
		a.emit("scan:progress", ScanProgressDTO{
			DomainsScanned: totalSession,
			SessionScanned: sessionScanned,
			WaveScanned:    chunkScanned,
			VulnsFound:     vulns,
			DomainsPending: centralPending,
			WaveBatchSize:  chunkBatchSize,
			Running:        active > 0,
		})
	}

	err := scanorch.Run(scanorch.Options{
		Ctx: ctx, AppRoot: a.repoRoot, DataRoot: a.dataRoot, ScanDir: dir,
		DBPath: a.dbPath, FindingsDir: a.findingsDir,
		LocalVersion: paths.InstallVersion(a.repoRoot), RunID: runID,
		IncludeLocal: includeLocal, RemoteWorkers: remoteWorkers,
		DeployBefore: opts.DeployRemote, DeployRepo: userDeployRepo(), HubEnabled: user.HubEnabledOrDefault(),
		Threads: threads, PathWorkers: pathWorkers, Fast: opts.Fast, Rescan: opts.Rescan, TimeoutSec: timeout,
		Findings: a.findings, Domains: a.domainStore,
		OnOutput: func(line string) { a.emit("scan:output", line) },
		OnWorkerChunkComplete: func(_ string, chunkSize int, _ int64) {
			progressMu.Lock()
			sessionScanned += int64(chunkSize)
			chunkBatchSize = chunkSize
			progressMu.Unlock()
			emitAggregate()
		},
		OnWorkerProgress: func(p scanorch.WorkerProgress) {
			progressMu.Lock()
			if prev, ok := workerTotals[p.WorkerID]; ok {
				newChunk := p.Status == "preparing" ||
					(p.DomainsScanned == 0 && prev.DomainsTotal > 0 && prev.DomainsScanned >= prev.DomainsTotal)
				if !newChunk && p.DomainsTotal > 0 && p.DomainsTotal == prev.DomainsTotal {
					if p.DomainsScanned < prev.DomainsScanned {
						p.DomainsScanned = prev.DomainsScanned
					}
					if p.VulnsFound < prev.VulnsFound {
						p.VulnsFound = prev.VulnsFound
					}
				}
			}
			workerTotals[p.WorkerID] = p
			progressMu.Unlock()
			a.emit("scan:worker-progress", ScanWorkerProgressDTO{
				WorkerID: p.WorkerID, WorkerName: p.WorkerName,
				DomainsScanned: p.DomainsScanned, VulnsFound: p.VulnsFound,
				DomainsTotal: p.DomainsTotal, Status: p.Status, Error: p.Error, Running: p.Running,
				PhasePercent: p.PhasePercent, PhaseLabel: p.PhaseLabel,
			})
			emitAggregate()
		},
		OnFound: func(workerID, domain, path, url string, isNew bool) {
			if isNew {
				progressMu.Lock()
				sessionVulns++
				progressMu.Unlock()
			}
			a.emit("scan:found", map[string]any{
				"domain": domain, "url": url, "path": path, "workerId": workerID, "isNew": isNew,
			})
			emitAggregate()
		},
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.emit("scan:output", "Scan cancelado.")
		} else {
			a.emit("scan:output", "Erro: "+err.Error())
		}
	} else {
		a.emit("scan:output", "Scan concluído.")
	}
	a.emit("scan:findings-refresh", struct{}{})
}

func userDeployRepo() settings.DeployRepo {
	user, _ := settings.Load()
	return user.DeployRepo.Normalized()
}

func (a *App) finishScan() {
	a.scanMu.Lock()
	a.scanCancel = nil
	a.scanMu.Unlock()
	a.emit("scan:progress", ScanProgressDTO{Running: false})
}

func (a *App) CancelScan() {
	a.scanMu.Lock()
	cancel := a.scanCancel
	a.scanCancel = nil
	a.scanMu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	a.emit("scan:progress", ScanProgressDTO{Running: false})
	a.emit("scan:output", "A parar scan…")
}

func (a *App) Shutdown() {
	a.CancelBatchCheck()
	a.CancelScan()
	if a.domainStore != nil {
		a.domainStore.Close()
	}
}
