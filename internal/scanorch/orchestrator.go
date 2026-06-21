package scanorch

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"goscan/internal/paths"
	"goscan/internal/remoteworker"
	"goscan/internal/scanhub"
	"goscan/internal/scanner"
	"goscan/internal/settings"
	"goscan/internal/store"
)

const localWorkerID = "local"

// DefaultWaveBatchSize is the target domain throughput per round across all workers combined.
const DefaultWaveBatchSize = 5000

// MinWorkerChunkSize is the minimum domains fetched per worker per round-trip.
const MinWorkerChunkSize = 500

func workerChunkSize(workerCount int) int {
	if workerCount <= 0 {
		return DefaultWaveBatchSize
	}
	n := DefaultWaveBatchSize / workerCount
	if n < MinWorkerChunkSize {
		n = MinWorkerChunkSize
	}
	return n
}

type waveChunk struct {
	domains []string
	dir     string
	cleanup func()
	count   int
}

// WorkerProgress reports per-worker scan state for the UI.
type WorkerProgress struct {
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

// Options configures a multi-worker scan run.
type Options struct {
	Ctx           context.Context
	AppRoot       string
	DataRoot      string
	ScanDir       string
	DBPath        string
	FindingsDir   string
	LocalVersion  string
	RunID         string
	IncludeLocal  bool
	RemoteWorkers []settings.RemoteWorker
	DeployBefore  bool
	DeployRepo    settings.DeployRepo
	HubEnabled    bool
	Threads       int
	PathWorkers   int
	Fast          bool
	Rescan        bool
	TimeoutSec    int

	Findings *store.FindingsStore
	Domains  *store.DomainStore

	OnWorkerProgress      func(WorkerProgress)
	OnWorkerChunkComplete func(workerID string, chunkSize int, centralPending int64)
	OnFound               func(workerID, domain, path, url string, isNew bool)
	OnOutput         func(line string)
}

// Run orchestrates local and remote scans; each worker pulls the next chunk independently.
func Run(opts Options) error {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	if opts.RunID == "" {
		opts.RunID = paths.NewBatchRunID()
	}
	if opts.LocalVersion == "" {
		opts.LocalVersion = paths.InstallVersion(opts.AppRoot)
	}

	emitOut(opts, "A verificar domínios pendentes na base…")
	if err := opts.Ctx.Err(); err != nil {
		return err
	}

	added, err := syncDomainsFromFiles(opts, true)
	if err != nil {
		return fmt.Errorf("importar domínios: %w", err)
	}
	if added > 0 {
		emitOut(opts, fmt.Sprintf("+%d domínios novos na base central", added))
	}

	pendingTotal := opts.Domains.CountPending(opts.Rescan)
	if pendingTotal == 0 {
		emitOut(opts, "Nenhum domínio pendente para scan.")
		return nil
	}

	workerCount := len(opts.RemoteWorkers)
	if opts.IncludeLocal {
		workerCount++
	}
	if workerCount == 0 {
		return fmt.Errorf("nenhum destino seleccionado")
	}
	chunkSize := workerChunkSize(workerCount)

	emitOut(opts, fmt.Sprintf(
		"Scan %s — fila central: %d pendentes · lote %d dom/filho",
		opts.RunID, pendingTotal, chunkSize,
	))
	emitOut(opts, "Filhos independentes: cada um pede o próximo lote à central assim que termina (partição estável por rowid)")

	type job struct {
		id, name     string
		worker       settings.RemoteWorker
		local        bool
		workerIndex  int
	}
	var jobs []job
	idx := 0
	if opts.IncludeLocal {
		jobs = append(jobs, job{id: localWorkerID, name: "Local", local: true, workerIndex: idx})
		idx++
	}
	for _, w := range opts.RemoteWorkers {
		jobs = append(jobs, job{id: w.ID, name: w.Name, worker: w, workerIndex: idx})
		idx++
	}
	for _, j := range jobs {
		partPending := opts.Domains.CountWorkerChunk(j.workerIndex, workerCount, opts.Rescan)
		emitOut(opts, fmt.Sprintf("  · %s: ~%d dom na partição", j.name, partPending))
	}

	var hubRegistry *scanhub.Registry
	if len(opts.RemoteWorkers) > 0 && opts.HubEnabled {
		reg, err := scanhub.StartRegistry(opts.Ctx, scanhub.Handlers{
			OnProgress: func(p scanhub.ProgressEvent) {
				if opts.OnWorkerProgress == nil {
					return
				}
				opts.OnWorkerProgress(WorkerProgress{
					WorkerID: p.WorkerID, WorkerName: workerName(opts, p.WorkerID),
					DomainsScanned: p.Scanned, VulnsFound: p.Vulns,
					DomainsTotal: p.Total, Status: "running", Running: true,
				})
			},
			OnFound: func(ev scanhub.FoundEvent) {
				runID := opts.RunID + "-" + ev.WorkerID
				_, _, isNew, err := opts.Findings.SaveFinding(
					ev.Domain, ev.Path, ev.URL, ev.Confidence, runID, ev.Content, ev.HasCredentials,
				)
				if err != nil {
					emitOut(opts, fmt.Sprintf("[hub] erro ao guardar %s%s: %v", ev.Domain, ev.Path, err))
					return
				}
				name := workerName(opts, ev.WorkerID)
				if isNew {
					emitOut(opts, fmt.Sprintf("[%s] hub: novo finding %s%s", name, ev.Domain, ev.Path))
				} else {
					emitOut(opts, fmt.Sprintf("[%s] hub: finding já na base %s%s (conteúdo igual)", name, ev.Domain, ev.Path))
				}
				if opts.OnFound != nil {
					opts.OnFound(ev.WorkerID, ev.Domain, ev.Path, ev.URL, isNew)
				}
			},
		})
		if err != nil {
			emitOut(opts, "Aviso: hub local indisponível — scan remoto usa fallback stderr")
		} else {
			hubRegistry = reg
			defer hubRegistry.Close()
			emitOut(opts, "Scan hub activo (WebSocket + conteúdo .env encriptado)")
		}
	}

	updateProgress := func(p WorkerProgress) {
		if opts.OnWorkerProgress != nil {
			opts.OnWorkerProgress(p)
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(jobs))
	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			updateProgress(WorkerProgress{
				WorkerID: j.id, WorkerName: j.name, Status: "preparing", Running: true,
			})
			errCh <- runWorkerLoop(opts, j, hubRegistry, workerCount, chunkSize, updateProgress)
		}()
	}
	wg.Wait()
	close(errCh)

	var errs []string
	for err := range errCh {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		if opts.Ctx.Err() != nil {
			return opts.Ctx.Err()
		}
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	remaining := opts.Domains.CountPending(opts.Rescan)
	if remaining == 0 {
		emitOut(opts, "Scan completo — fila esgotada.")
	} else {
		emitOut(opts, fmt.Sprintf("Scan terminado — %d pendentes na fila central", remaining))
	}
	return nil
}

func syncDomainsFromFiles(opts Options, verbose bool) (int64, error) {
	files := scanner.FindInputFiles(opts.ScanDir)
	if len(files) == 0 {
		if verbose {
			return 0, fmt.Errorf("nenhum ficheiro de entrada em %s", opts.ScanDir)
		}
		return 0, nil
	}
	before := opts.Domains.Count()
	if before == 0 && verbose {
		emitOut(opts, fmt.Sprintf("A importar domínios de %d ficheiro(s) em batches de %d…", len(files), scanner.ImportBatchSize))
	}
	imported, err := scanner.ImportDomainsFromFilesCtx(opts.Ctx, files, opts.Domains, func(n int64, label string) {
		if !verbose {
			return
		}
		if n > 10000 && n%100000 != 0 {
			return
		}
		emitOut(opts, fmt.Sprintf("Importação: %d domínios únicos · %s", n, label))
	})
	if err != nil {
		return 0, err
	}
	after := opts.Domains.Count()
	added := after - before
	if verbose && before == 0 {
		emitOut(opts, fmt.Sprintf("Importação concluída: %d domínios únicos na base", imported))
	} else if verbose || added > 0 {
		emitOut(opts, fmt.Sprintf("Base: %d total · %d pendentes", after, opts.Domains.CountPending(opts.Rescan)))
	}
	return added, nil
}

func runWorkerLoop(opts Options, j struct {
	id, name     string
	worker       settings.RemoteWorker
	local        bool
	workerIndex  int
}, hub *scanhub.Registry, workerCount, chunkSize int, update func(WorkerProgress)) error {
	name := j.name
	chunkNum := 0
	var sessionDone int64
	deployed := false

	var sshSess *remoteworker.WorkerSession
	if !j.local && j.worker.ExecMode != settings.ExecHTTP {
		cfg := remoteworker.ConfigFrom(j.worker, opts.AppRoot, opts.LocalVersion, opts.DeployRepo)
		log := func(msg string) { emitOut(opts, fmt.Sprintf("[%s] %s", name, msg)) }
		uploadProgress := func(done, total int64, label string) {
			pct := 0
			if total > 0 {
				pct = int(done * 100 / total)
			}
			update(WorkerProgress{
				WorkerID: j.id, WorkerName: name, Status: "deploying", Running: true,
				PhasePercent: pct, PhaseLabel: label,
			})
		}
		var err error
		sshSess, err = remoteworker.ConnectWorkerSession(opts.Ctx, cfg, opts.DeployBefore, log, uploadProgress)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		defer sshSess.Close()
		if opts.DeployBefore {
			deployed = true
		}
	}

	for {
		if err := opts.Ctx.Err(); err != nil {
			update(WorkerProgress{WorkerID: j.id, WorkerName: name, Status: "cancelled", Running: false})
			return err
		}

		batch, err := opts.Domains.FetchWorkerChunk(j.workerIndex, workerCount, chunkSize, opts.Rescan)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			if sessionDone > 0 {
				emitOut(opts, fmt.Sprintf("[%s] partição esgotada — %d dom nesta sessão", name, sessionDone))
			}
			update(WorkerProgress{
				WorkerID: j.id, WorkerName: name,
				DomainsScanned: sessionDone, DomainsTotal: sessionDone,
				Status: "done", Running: false,
			})
			return nil
		}

		chunkNum++
		pending := opts.Domains.CountPending(opts.Rescan)
		emitOut(opts, fmt.Sprintf("[%s] lote %d · %d dom · %d pendentes na fila central", name, chunkNum, len(batch), pending))

		dir, cleanup, err := WriteChunkDir(batch)
		if err != nil {
			return err
		}
		chunk := waveChunk{domains: batch, dir: dir, cleanup: cleanup, count: len(batch)}

		update(WorkerProgress{
			WorkerID: j.id, WorkerName: name,
			DomainsScanned: 0, DomainsTotal: int64(len(batch)),
			Status: "preparing", Running: true,
		})

		var jobErr error
		if j.local {
			jobErr = runLocalJob(opts, j.id, chunk, update)
		} else {
			deploy := opts.DeployBefore && !deployed
			jobErr = runRemoteJob(opts, hub, sshSess, j.worker, j.id, name, chunk, deploy, update)
			if jobErr == nil && deploy {
				deployed = true
			}
		}
		cleanup()

		if jobErr != nil {
			update(WorkerProgress{WorkerID: j.id, WorkerName: name, Status: "failed", Error: jobErr.Error(), Running: false})
			return jobErr
		}

		emitOut(opts, fmt.Sprintf("[%s] a marcar %d dom na base central…", name, len(batch)))
		markErr := opts.Domains.MarkScannedList(batch)
		if markErr != nil {
			return markErr
		}
		sessionDone += int64(len(batch))

		remaining := opts.Domains.CountPending(opts.Rescan)
		emitOut(opts, fmt.Sprintf("[%s] lote %d concluído · %d nesta sessão · %d pendentes central · a pedir próximo…", name, chunkNum, sessionDone, remaining))
		if opts.OnWorkerChunkComplete != nil {
			opts.OnWorkerChunkComplete(j.id, len(batch), remaining)
		}
		update(WorkerProgress{
			WorkerID: j.id, WorkerName: name,
			DomainsScanned: int64(len(batch)), DomainsTotal: int64(len(batch)),
			Status: "running", Running: true, PhaseLabel: "próximo lote",
		})
	}
}

func runLocalJob(opts Options, workerID string, chunk waveChunk, update func(WorkerProgress)) error {
	if err := opts.Ctx.Err(); err != nil {
		update(WorkerProgress{WorkerID: workerID, WorkerName: "Local", Status: "cancelled", Running: false})
		return err
	}
	count := chunk.count
	emitOut(opts, fmt.Sprintf("[Local] lote: %d domínios", count))

	runID := opts.RunID + "-" + workerID
	timeout := opts.TimeoutSec
	if timeout <= 0 {
		timeout = 8
	}
	var lastLog int64
	cfg := &scanner.Config{
		RepoRoot:      opts.DataRoot,
		Dir:           chunk.dir,
		DBPath:        opts.DBPath,
		FindingsDir:   opts.FindingsDir,
		FindingsStore: opts.Findings,
		RunID:         runID,
		Threads:     opts.Threads,
		PathWorkers: opts.PathWorkers,
		Fast:        opts.Fast,
		SaveContent: true,
		Timeout:     time.Duration(timeout) * time.Second,
		OnProgress: func(s scanner.Stats) {
			total := int64(count)
			update(WorkerProgress{
				WorkerID: workerID, WorkerName: "Local",
				DomainsScanned: s.DomainsScanned, VulnsFound: s.VulnsFound,
				DomainsTotal: total, Status: "running", Running: true,
			})
			if s.DomainsScanned/250 > lastLog/250 || s.DomainsScanned >= total {
				lastLog = s.DomainsScanned
				emitOut(opts, fmt.Sprintf("[Local] scan: %d/%d dom · %d vulns", s.DomainsScanned, total, s.VulnsFound))
			}
		},
		OnFound: func(r scanner.VulnResult) {
			emitOut(opts, fmt.Sprintf("[Local] FOUND %s%s", r.Domain, r.Path))
			if opts.OnFound != nil {
				opts.OnFound(workerID, r.Domain, r.Path, r.URL, true)
			}
		},
	}
	scanCtx, scanCancel := context.WithCancel(opts.Ctx)
	defer scanCancel()
	if err := scanner.RunChunkScan(scanCtx, cfg); err != nil {
		update(WorkerProgress{WorkerID: workerID, WorkerName: "Local", Status: "failed", Error: err.Error(), Running: false})
		return err
	}
	return nil
}

func workerName(opts Options, workerID string) string {
	if workerID == localWorkerID {
		return "Local"
	}
	for _, w := range opts.RemoteWorkers {
		if w.ID == workerID {
			return w.Name
		}
	}
	return workerID
}

func runRemoteJob(opts Options, hub *scanhub.Registry, sshSess *remoteworker.WorkerSession, w settings.RemoteWorker, workerID, name string, chunk waveChunk, deployBefore bool, update func(WorkerProgress)) error {
	if err := opts.Ctx.Err(); err != nil {
		update(WorkerProgress{WorkerID: workerID, WorkerName: name, Status: "cancelled", Running: false})
		return err
	}
	count := chunk.count
	cfg := remoteworker.ConfigFrom(w, opts.AppRoot, opts.LocalVersion, opts.DeployRepo)
	log := func(msg string) { emitOut(opts, fmt.Sprintf("[%s] %s", name, msg)) }
	uploadProgress := func(done, total int64, label string) {
		pct := 0
		if total > 0 {
			pct = int(done * 100 / total)
		}
		update(WorkerProgress{
			WorkerID: workerID, WorkerName: name, Status: "deploying", Running: true,
			PhasePercent: pct, PhaseLabel: label,
		})
	}

	log(fmt.Sprintf("lote: %d domínios — a enviar ao filho…", count))
	dir := chunk.dir

	runID := opts.RunID + "-" + workerID
	update(WorkerProgress{
		WorkerID: workerID, WorkerName: name, DomainsTotal: int64(count),
		Status: "running", Running: true,
	})

	scanOpts := remoteworker.ScanOptions{
		RunID:            runID,
		MasterRunID:      opts.RunID,
		WorkerID:         workerID,
		ChunkDir:         dir,
		DomainCount:      count,
		Threads:          opts.Threads,
		PathWorkers:      opts.PathWorkers,
		Fast:             opts.Fast,
		TimeoutSec:       opts.TimeoutSec,
		DeployBefore:     deployBefore,
		OnUploadProgress: uploadProgress,
	}
	hubLabel := "legacy"
	var hubConnected atomic.Bool
	if hub != nil {
		token, err := hub.RegisterWorker(workerID, runID, int64(count))
		if err == nil {
			scanOpts.Hub = &remoteworker.HubAttach{
				LocalAddr: hub.LocalAddr(),
				Token:     token,
			}
			scanOpts.HubConnected = &hubConnected
			hubLabel = "fallback"
		} else {
			log("hub: falha ao registar token — fallback stderr/export (" + err.Error() + ")")
		}
	}

	scanOpts.OnScanProgress = func(scanned, vulns, total int64) {
		if hubConnected.Load() {
			return
		}
		label := hubLabel
		if hub != nil && hubConnected.Load() {
			label = "hub"
		}
		update(WorkerProgress{
			WorkerID: workerID, WorkerName: name,
			DomainsScanned: scanned, VulnsFound: vulns,
			DomainsTotal: total, Status: "running", Running: true, PhaseLabel: label,
		})
	}
	scanOpts.OnFound = func(domain, path, url string) {
		if opts.OnFound != nil {
			opts.OnFound(workerID, domain, path, url, true)
		}
	}

	var raw []byte
	var err error
	switch w.ExecMode {
	case settings.ExecHTTP:
		raw, err = remoteworker.RunHTTPScan(opts.Ctx, cfg, scanOpts, func(p remoteworker.HTTPProgress) {
			update(WorkerProgress{
				WorkerID: workerID, WorkerName: name,
				DomainsScanned: p.DomainsScanned, VulnsFound: p.VulnsFound,
				DomainsTotal: int64(count), Status: "running", Running: p.Running,
			})
		}, log)
	default:
		if sshSess != nil {
			raw, err = sshSess.RunBatch(opts.Ctx, scanOpts, log)
		} else {
			raw, err = remoteworker.RunSSHScan(opts.Ctx, cfg, scanOpts, log)
		}
	}
	if err != nil {
		update(WorkerProgress{WorkerID: workerID, WorkerName: name, Status: "failed", Error: err.Error()})
		return fmt.Errorf("%s: %w", name, err)
	}

	log("a fundir findings na base local…")
	imported, err := opts.Findings.ImportFindingsJSON(bytes.NewReader(raw), workerID, opts.RunID)
	if err != nil {
		log(fmt.Sprintf("erro a importar findings: %v", err))
		update(WorkerProgress{WorkerID: workerID, WorkerName: name, Status: "failed", Error: err.Error()})
		return fmt.Errorf("%s import: %w", name, err)
	}
	log(fmt.Sprintf("export reconciliado: %d findings", imported))
	return nil
}

func emitOut(opts Options, line string) {
	if opts.OnOutput != nil {
		opts.OnOutput(line)
	}
}
