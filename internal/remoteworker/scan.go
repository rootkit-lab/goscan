package remoteworker

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// HubAttach configures WebSocket streaming to the local orchestrator hub.
type HubAttach struct {
	LocalAddr string // orchestrator hub listen addr (127.0.0.1:PORT)
	WSURL     string
	Token     string
}

// ScanOptions configures a remote scan job.
type ScanOptions struct {
	RunID            string
	MasterRunID      string
	WorkerID         string
	ChunkDir         string
	DomainCount      int
	Threads          int
	PathWorkers      int
	Fast             bool
	TimeoutSec       int
	RemoteBin        string // preenchido após ensureInstalledOnClient ou WorkerSession
	DeployBefore     bool
	Hub              *HubAttach
	HubConnected     *atomic.Bool
	OnUploadProgress UploadProgress
	OnScanProgress   func(scanned, vulns, total int64)
	OnFound          func(domain, path, url string)
}

func runSSHBatch(ctx context.Context, client *ssh.Client, w Config, opts ScanOptions, onLog func(string)) ([]byte, error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	if opts.RunID == "" {
		return nil, fmt.Errorf("run-id em falta")
	}
	if opts.ChunkDir == "" {
		return nil, fmt.Errorf("chunk dir em falta")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	goscanBin := opts.RemoteBin
	if goscanBin == "" {
		var err error
		goscanBin, err = ensureInstalledOnClient(client, w, opts.DeployBefore, onLog, opts.OnUploadProgress)
		if err != nil {
			return nil, err
		}
	}

	runSlug := sanitizeRunID(opts.RunID)
	remoteChunk := fmt.Sprintf("/tmp/goscan-chunk-%s", runSlug)
	remoteRun := fmt.Sprintf("/tmp/goscan-run-%s", runSlug)
	defer func() {
		killRemoteScan(context.Background(), client, opts.RunID)
		cleanupRemote(client, remoteChunk, remoteRun)
	}()

	onLog(fmt.Sprintf("a enviar batch de %d domínios únicos…", opts.DomainCount))
	if err := uploadTree(ctx, client, opts.ChunkDir, remoteChunk, onLog, opts.OnUploadProgress); err != nil {
		return nil, fmt.Errorf("upload batch: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
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

	setup := fmt.Sprintf(`mkdir -p %s/findings/by-domain`, shellQuote(remoteRun))
	if _, err := runSession(client, setup); err != nil {
		return nil, err
	}

	remoteFindings := remoteRun + "/findings"

	goscanBin = strings.TrimSpace(goscanBin)
	if goscanBin == "" {
		return nil, fmt.Errorf("binário remoto em falta — instalação não concluída")
	}
	if !remoteExecutable(client, goscanBin) {
		return nil, fmt.Errorf("goscan não executável em %s — activa Deploy ou verifica instalação", goscanBin)
	}

	const remoteHubPort = "127.0.0.1:19280"
	hubFlags := ""
	var tunnel *Tunnel
	if opts.Hub != nil && opts.Hub.LocalAddr != "" && opts.Hub.Token != "" {
		onLog("a abrir túnel hub (WebSocket seguro)…")
		var err error
		tunnel, err = OpenTunnel(ctx, client, remoteHubPort, opts.Hub.LocalAddr)
		if err != nil {
			onLog("túnel hub falhou — fallback stderr")
		} else {
			defer tunnel.Close()
			hubFlags = fmt.Sprintf(
				" -hub %s -hub-token %s -worker-id %s",
				shellQuote("ws://"+remoteHubPort+"/hub"),
				shellQuote(opts.Hub.Token),
				shellQuote(opts.WorkerID),
			)
			onLog("hub WebSocket pronto")
		}
	}

	progressJSON := ""
	if hubFlags == "" {
		progressJSON = " -progress-json"
	}

	scanCmd := fmt.Sprintf(
		`GOSCAN_MODE=prod %s -dir %s -findings %s -run-id %s -batch-size %d%s -threads %d -path-workers %d -timeout %d %s%s 2>&1`,
		shellQuote(goscanBin),
		shellQuote(remoteChunk),
		shellQuote(remoteFindings),
		shellQuote(opts.RunID),
		opts.DomainCount,
		progressJSON,
		threads,
		pathWorkers,
		timeout,
		boolFlag("-fast", opts.Fast),
		hubFlags,
	)
	onLog(fmt.Sprintf("a testar batch (%d threads, %d domínios)…", threads, opts.DomainCount))
	var lastProgress int64
	hubActive := opts.HubConnected
	if err := runSessionStream(ctx, client, scanCmd, func(line string) {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\r"))
		if line == "" {
			return
		}
		if scanned, vulns, total, ok := parseProgressLine(line); ok {
			if hubActive == nil || !hubActive.Load() {
				if opts.OnScanProgress != nil {
					opts.OnScanProgress(scanned, vulns, total)
				}
				if scanned/250 > lastProgress/250 || scanned >= total {
					lastProgress = scanned
					onLog(fmt.Sprintf("scan: %d/%d dom · %d vulns", scanned, total, vulns))
				}
			}
			return
		}
		if domain, path, url, ok := parseFoundLine(line); ok {
			if opts.OnFound != nil {
				opts.OnFound(domain, path, url)
			}
			return
		}
		if strings.HasPrefix(line, "@goscan/hub connected") {
			if opts.HubConnected != nil {
				opts.HubConnected.Store(true)
			}
			onLog("hub conectado no filho")
			return
		}
		if strings.HasPrefix(line, "@goscan/") {
			return
		}
		if strings.Contains(line, "⏳") || strings.Contains(line, "linhas |") {
			return
		}
		onLog(line)
	}); err != nil {
		return nil, fmt.Errorf("scan remoto: %w", err)
	}

	onLog("a devolver findings ao orchestrador…")
	exportCmd := fmt.Sprintf(
		`%s findings export-json -run-id %s -findings %s`,
		shellQuote(goscanBin),
		shellQuote(opts.RunID),
		shellQuote(remoteFindings),
	)
	out, err := runSession(client, exportCmd)
	if err != nil {
		return nil, fmt.Errorf("export findings: %w", err)
	}
	return []byte(out), nil
}

func cleanupRemote(client *ssh.Client, paths ...string) {
	for _, p := range paths {
		_, _ = runSession(client, "rm -rf "+shellQuote(p))
	}
}

func boolFlag(name string, on bool) string {
	if on {
		return name
	}
	return ""
}

func sanitizeRunID(id string) string {
	id = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, id)
	if id == "" {
		return fmt.Sprintf("run-%d", time.Now().Unix())
	}
	return id
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\"'\"'`) + "'"
}
