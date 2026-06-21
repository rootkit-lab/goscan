package remoteworker

import (
	"context"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// WorkerSession keeps one SSH connection and remote binary path across batch scans.
type WorkerSession struct {
	client *ssh.Client
	bin    string
	cfg    Config
}

// ConnectWorkerSession dials SSH once and ensures goscan-remote is installed.
func ConnectWorkerSession(ctx context.Context, cfg Config, deployBefore bool, onLog func(string), onProgress UploadProgress) (*WorkerSession, error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	onLog("a ligar SSH (sessão persistente)…")
	client, err := dial(cfg)
	if err != nil {
		return nil, err
	}
	bin, err := ensureInstalledOnClient(client, cfg, deployBefore, onLog, onProgress)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &WorkerSession{client: client, bin: bin, cfg: cfg}, nil
}

// Close closes the SSH session.
func (s *WorkerSession) Close() {
	if s != nil && s.client != nil {
		s.client.Close()
		s.client = nil
	}
}

// RunSSHScan is a one-shot scan (dials, runs one batch, closes).
func RunSSHScan(ctx context.Context, w Config, opts ScanOptions, onLog func(string)) ([]byte, error) {
	sess, err := ConnectWorkerSession(ctx, w, opts.DeployBefore, onLog, opts.OnUploadProgress)
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	return sess.RunBatch(ctx, opts, onLog)
}

// RunBatch scans one uploaded chunk using the persistent SSH session.
func (s *WorkerSession) RunBatch(ctx context.Context, opts ScanOptions, onLog func(string)) ([]byte, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("sessão SSH fechada")
	}
	opts.RemoteBin = s.bin
	return runSSHBatch(ctx, s.client, s.cfg, opts, onLog)
}
