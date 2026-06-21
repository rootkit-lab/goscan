package remoteworker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func runSessionWithContext(ctx context.Context, client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()
	select {
	case <-ctx.Done():
		_ = session.Close()
		<-done
		return ctx.Err()
	case err := <-done:
		_ = session.Close()
		return err
	}
}

func uploadTree(ctx context.Context, client *ssh.Client, localDir, remoteDir string, onLog func(string), onProgress UploadProgress) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	total, err := dirByteSize(localDir)
	if err != nil {
		return err
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	var transferred int64
	lastPct := -1
	step := func(n int64) {
		transferred += n
		uploadProgressStep(&lastPct, transferred, total, "envio batch", onLog, onProgress)
	}
	if err := uploadDirCtx(ctx, sftpClient, localDir, remoteDir, step); err != nil {
		return err
	}
	return nil
}

func uploadDirCtx(ctx context.Context, client *sftp.Client, localDir, remoteDir string, step func(int64)) error {
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		rpath := filepath.ToSlash(filepath.Join(remoteDir, rel))
		if info.IsDir() {
			_ = sftpMkdirAll(client, rpath)
			return nil
		}
		st, err := os.Stat(path)
		if err != nil {
			return err
		}
		var prev int64
		fileStep := func(done, _ int64) {
			if step != nil && done > prev {
				step(done - prev)
				prev = done
			}
		}
		return uploadFileWithProgress(client, path, rpath, st.Size(), fileStep)
	})
}

func killRemoteScan(ctx context.Context, client *ssh.Client, runID string) {
	if client == nil {
		return
	}
	slug := sanitizeRunID(runID)
	_, _ = runSession(client, fmt.Sprintf(
		`pkill -f "goscan.*%s" 2>/dev/null; rm -rf %s %s 2>/dev/null; true`,
		slug,
		shellQuote("/tmp/goscan-chunk-"+slug),
		shellQuote("/tmp/goscan-run-"+slug),
	))
	_ = ctx
}
