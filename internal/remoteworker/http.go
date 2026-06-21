package remoteworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// HTTPProgress mirrors worker API progress.
type HTTPProgress struct {
	DomainsScanned int64
	VulnsFound     int64
	Running        bool
}

// RunHTTPScan uploads a batch and runs an ephemeral HTTP worker scan on the remote host.
func RunHTTPScan(ctx context.Context, w Config, opts ScanOptions, onProgress func(HTTPProgress), onLog func(string)) ([]byte, error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	onLog("a ligar SSH…")
	client, err := dial(w)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	goscanBin, err := ensureInstalledOnClient(client, w, opts.DeployBefore, onLog, opts.OnUploadProgress)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	port := w.APIPort
	if port <= 0 {
		port = 9090
	}
	token := w.APIToken
	startWorker := fmt.Sprintf(
		`nohup %s worker -listen :%d -token %s >/tmp/goscan-worker.log 2>&1 & sleep 1`,
		shellQuote(goscanBin), port, shellQuote(token),
	)
	if _, err := runSession(client, startWorker); err != nil {
		return nil, fmt.Errorf("iniciar worker HTTP: %w", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	reqBody, _ := json.Marshal(map[string]any{
		"dir":         remoteChunk,
		"runId":       opts.RunID,
		"threads":     opts.Threads,
		"pathWorkers": opts.PathWorkers,
		"fast":        opts.Fast,
		"timeoutSec":  opts.TimeoutSec,
		"ephemeral":   true,
	})
	if _, err := remoteCurl(client, "POST", baseURL+"/v1/scan", token, string(reqBody)); err != nil {
		return nil, err
	}

	onLog("a testar batch via HTTP…")
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		out, err := remoteCurl(client, "GET", baseURL+"/v1/progress", token, "")
		if err != nil {
			return nil, err
		}
		var body struct {
			Running        bool  `json:"running"`
			DomainsScanned int64 `json:"domainsScanned"`
			VulnsFound     int64 `json:"vulnsFound"`
		}
		if err := json.Unmarshal([]byte(out), &body); err != nil {
			return nil, err
		}
		if onProgress != nil {
			onProgress(HTTPProgress{
				DomainsScanned: body.DomainsScanned,
				VulnsFound:     body.VulnsFound,
				Running:        body.Running,
			})
		}
		if !body.Running {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	onLog("a devolver findings ao orchestrador…")
	exportURL := baseURL + "/v1/findings/export?runId=" + url.QueryEscape(opts.RunID)
	out, err := remoteCurl(client, "GET", exportURL, token, "")
	if err != nil {
		return nil, err
	}
	cleanupRemote(client, remoteChunk, remoteRun)
	onLog("batch concluído — dados temporários removidos no servidor")
	return []byte(out), nil
}

func remoteCurl(client *ssh.Client, method, rawURL, token, body string) (string, error) {
	parts := []string{"curl", "-sS", "-X", method}
	if body != "" {
		parts = append(parts, "-H", shellQuote("Content-Type: application/json"), "-d", shellQuote(body))
	}
	if token != "" {
		parts = append(parts, "-H", shellQuote("Authorization: Bearer "+token))
	}
	parts = append(parts, shellQuote(rawURL))
	cmd := strings.Join(parts, " ")
	out, err := runSession(client, cmd)
	if err != nil {
		return "", fmt.Errorf("curl remoto: %w", err)
	}
	return out, nil
}
