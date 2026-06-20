package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"goscan/internal/paths"
	"goscan/internal/scanner"
	"goscan/internal/scripts"
	"goscan/internal/store"
)

type App struct {
	wails       *application.App
	repoRoot    string
	dbPath      string
	findingsDir string
	domainStore *store.DomainStore
	findings    *store.FindingsStore
	scriptRun   *scripts.Runner

	scanMu     sync.Mutex
	scanCancel context.CancelFunc
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

func NewApp() (*App, error) {
	root, err := paths.RepoRoot()
	if err != nil {
		return nil, err
	}
	dbPath := paths.DefaultDBPath(root)
	findingsDir := paths.FindingsRoot(root)

	domainStore, err := store.OpenDomainStore(dbPath)
	if err != nil {
		return nil, err
	}
	fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
	if err != nil {
		domainStore.Close()
		return nil, err
	}
	runner, err := scripts.NewRunner(root)
	if err != nil {
		domainStore.Close()
		return nil, err
	}

	return &App{
		repoRoot:    root,
		dbPath:      dbPath,
		findingsDir: findingsDir,
		domainStore: domainStore,
		findings:    fs,
		scriptRun:   runner,
	}, nil
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
	}
}

func (a *App) SearchFindings(query, confidence string, limit int) ([]FindingDTO, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := a.findings.Search(store.FindingsFilter{Query: query, Confidence: confidence, Limit: limit})
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

func (a *App) RunScript(scriptID string, findingID int64) error {
	detail, err := a.GetFinding(findingID)
	if err != nil {
		return err
	}
	s, err := a.scriptRun.Find(scriptID)
	if err != nil {
		return err
	}
	emit := func(event string, data any) { a.emit(event, data) }
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
		dir = filepath.Join(a.repoRoot, "files")
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
		RepoRoot:    a.repoRoot,
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
	a.CancelScan()
	if a.domainStore != nil {
		a.domainStore.Close()
	}
}
