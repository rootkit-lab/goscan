package remoteworker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const remoteAppDir = ".local/share/goscan/app"

var deployBinCache struct {
	mu      sync.Mutex
	key     string
	binPath string
	binSize int64
}

func remoteHashFile(home string) string {
	return filepath.ToSlash(filepath.Join(home, remoteAppDir, "BINHASH"))
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func remoteAppBin(home string) string {
	return filepath.ToSlash(filepath.Join(home, remoteAppDir, "bin", "goscan-remote"))
}

func remoteBinUpToDateSFTP(client *sftp.Client, home, localVer string, localSize int64, localHash string) (bool, string) {
	remoteBin := remoteAppBin(home)
	st, err := client.Stat(remoteBin)
	if err != nil || st.IsDir() || st.Size() != localSize {
		return false, "tamanho diferente"
	}
	rh := strings.TrimSpace(readRemoteText(client, remoteHashFile(home)))
	if rh == "" {
		return false, "sem BINHASH"
	}
	if rh != localHash {
		return false, "hash diferente"
	}
	rver := strings.TrimSpace(readRemoteText(client, filepath.ToSlash(filepath.Join(home, remoteAppDir, "VERSION"))))
	label := rver
	if label == "" {
		label = localVer
	}
	if label == "" {
		label = "ok"
	}
	return true, "v" + label + " · hash OK"
}

func remoteBinLink(home string) string {
	return filepath.ToSlash(filepath.Join(home, ".local", "bin", "goscan-remote"))
}

// EnsureInstalled guarantees goscan exists on the remote host.
// forceUpdate=true runs version/size check and may re-upload; false only installs if missing.
func EnsureInstalled(w Config, forceUpdate bool, onLog func(string), onProgress UploadProgress) (binPath string, err error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	client, err := dial(w)
	if err != nil {
		return "", err
	}
	defer client.Close()
	return ensureInstalledOnClient(client, w, forceUpdate, onLog, onProgress)
}

func ensureInstalledOnClient(client *ssh.Client, w Config, forceUpdate bool, onLog func(string), onProgress UploadProgress) (binPath string, err error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	home, err := remoteHome(client)
	if err != nil {
		return "", err
	}
	binPath = remoteAppBin(home)

	localBin, localVer, localSize, err := resolveDeployArtifacts(w.AppRoot, w.LocalVersion)
	if err != nil {
		return "", err
	}
	localHash, err := fileSHA256(localBin)
	if err != nil {
		return "", err
	}

	repo := w.DeployRepo.Normalized()
	useGit := repo.Enabled()

	if remoteExecutable(client, binPath) {
		sftpClient, sftpErr := sftp.NewClient(client)
		if sftpErr == nil {
			upToDate, reason := remoteBinUpToDateSFTP(sftpClient, home, localVer, localSize, localHash)
			if upToDate {
				sftpClient.Close()
				if forceUpdate {
					onLog("deploy ignorado (" + reason + ")")
				} else {
					onLog("goscan já instalado no filho (" + reason + ")")
				}
				return binPath, nil
			}
			if reason != "" {
				onLog("binário remoto desactualizado (" + reason + ") — a actualizar…")
			}
			sftpClient.Close()
		}
	} else if !forceUpdate {
		onLog("goscan em falta no filho — instalação automática…")
	} else {
		onLog("a verificar/atualizar goscan no filho…")
	}

	if useGit {
		if err := installFromGit(client, w, home, localVer, localHash, onLog); err != nil {
			return "", err
		}
	} else if err := uploadDeploy(client, w, home, localHash, onLog, onProgress); err != nil {
		return "", err
	}
	if !remoteExecutable(client, binPath) {
		return "", fmt.Errorf("goscan não encontrado após instalação (%s)", binPath)
	}
	onLog("binário pronto no filho")
	return binPath, nil
}

// Deploy uploads goscan if remote version/size differs (legacy API for explicit deploy).
func Deploy(w Config, onLog func(string)) (skipped bool, err error) {
	if onLog == nil {
		onLog = func(string) {}
	}
	client, err := dial(w)
	if err != nil {
		return false, err
	}
	defer client.Close()

	home, err := remoteHome(client)
	if err != nil {
		return false, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return false, err
	}
	defer sftpClient.Close()

	localBin, localVer, localSize, err := resolveDeployArtifacts(w.AppRoot, w.LocalVersion)
	if err != nil {
		return false, err
	}
	localHash, err := fileSHA256(localBin)
	if err != nil {
		return false, err
	}

	if upToDate, reason := remoteBinUpToDateSFTP(sftpClient, home, localVer, localSize, localHash); upToDate {
		onLog("deploy ignorado (" + reason + ")")
		return true, nil
	}

	if err := uploadDeploy(client, w, home, localHash, onLog, nil); err != nil {
		return false, err
	}
	onLog("deploy concluído")
	return false, nil
}

func uploadDeploy(client *ssh.Client, w Config, home, localHash string, onLog func(string), onProgress UploadProgress) error {
	localBin, localVer, localSize, err := resolveDeployArtifacts(w.AppRoot, w.LocalVersion)
	if err != nil {
		return err
	}
	if localHash == "" {
		localHash, err = fileSHA256(localBin)
		if err != nil {
			return err
		}
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	remoteBin := remoteAppBin(home)
	remoteVerFile := filepath.ToSlash(filepath.Join(home, remoteAppDir, "VERSION"))
	remoteHashPath := remoteHashFile(home)

	onLog("a enviar binário (~" + formatSize(localSize) + ")…")
	if err := sftpMkdirAll(sftpClient, filepath.ToSlash(filepath.Join(home, remoteAppDir, "bin"))); err != nil {
		return err
	}
	lastPct := -1
	progress := func(done, total int64) {
		uploadProgressStep(&lastPct, done, total, "envio binário", onLog, onProgress)
	}
	if err := uploadFileWithProgress(sftpClient, localBin, remoteBin, localSize, progress); err != nil {
		return fmt.Errorf("upload binário: %w", err)
	}
	if lastPct < 100 {
		progress(localSize, localSize)
	}
	if localVer != "" {
		if err := uploadBytes(sftpClient, []byte(localVer+"\n"), remoteVerFile); err != nil {
			return fmt.Errorf("upload VERSION: %w", err)
		}
	}
	if err := uploadBytes(sftpClient, []byte(localHash+"\n"), remoteHashPath); err != nil {
		return fmt.Errorf("upload BINHASH: %w", err)
	}

	onLog("a activar binário no PATH…")
	link := remoteBinLink(home)
	install := fmt.Sprintf(
		`set -e
mkdir -p "$HOME/.local/bin"
chmod +x %s
ln -sf %s %s
`,
		shellQuote(remoteBin), shellQuote(remoteBin), shellQuote(link),
	)
	if _, err := runSession(client, install); err != nil {
		return fmt.Errorf("install remoto: %w", err)
	}
	return nil
}

func remoteExecutable(client *ssh.Client, binPath string) bool {
	_, err := runSession(client, "test -x "+shellQuote(binPath))
	return err == nil
}

func resolveDeployArtifacts(appRoot, version string) (binPath, ver string, size int64, err error) {
	candidate := filepath.Join(appRoot, "bin", "goscan-remote")
	st, statErr := os.Stat(candidate)
	if statErr != nil || st.IsDir() {
		return "", "", 0, fmt.Errorf("bin/goscan-remote em falta em %s — corre make release && make install", appRoot)
	}
	binPath = candidate
	size = st.Size()
	ver = strings.TrimSpace(version)
	if ver == "" {
		if b, e := os.ReadFile(filepath.Join(appRoot, "assets", "VERSION")); e == nil {
			ver = strings.TrimSpace(string(b))
		}
	}
	key := ver + ":" + binPath + ":" + fmt.Sprintf("%d:%d", st.Size(), st.ModTime().Unix())
	deployBinCache.mu.Lock()
	deployBinCache.key = key
	deployBinCache.binPath = binPath
	deployBinCache.binSize = size
	deployBinCache.mu.Unlock()
	return binPath, ver, size, nil
}

func readRemoteText(client *sftp.Client, path string) string {
	f, err := client.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	return string(b)
}

func remoteHome(client *ssh.Client) (string, error) {
	// Sessões SSH não-interactivas (ex.: root) muitas vezes não definem $HOME.
	const script = `
home="${HOME:-}"
if [ -z "$home" ] || [ "$home" = '$HOME' ]; then
  home="$(getent passwd "$(id -un)" 2>/dev/null | cut -d: -f6)"
fi
if [ -z "$home" ]; then
  uid="$(id -un)"
  if [ "$uid" = root ]; then home=/root; else home="/home/$uid"; fi
fi
if [ ! -d "$home" ]; then
  home="$(cd ~ 2>/dev/null && pwd -P)" || true
fi
printf '%s' "$home"
`
	out, err := runSession(client, strings.TrimSpace(script))
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(out)
	home := ""
	for _, line := range strings.Split(raw, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "/") {
			home = l
			break
		}
	}
	if home == "" {
		// fallback seguro: assume root
		return "/root", nil
	}
	return home, nil
}

func sftpMkdirAll(client *sftp.Client, path string) error {
	parts := strings.Split(path, "/")
	cur := ""
	if strings.HasPrefix(path, "/") {
		cur = "/"
	}
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur = strings.TrimSuffix(cur, "/") + "/" + p
		_ = client.Mkdir(cur)
	}
	return nil
}

func uploadFileFast(client *sftp.Client, localPath, remotePath string) error {
	st, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	return uploadFileWithProgress(client, localPath, remotePath, st.Size(), nil)
}

func uploadFileWithProgress(client *sftp.Client, localPath, remotePath string, total int64, step func(done, total int64)) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	buf := make([]byte, uploadBufSize)
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if ew != nil {
				return ew
			}
			written += int64(nw)
			if step != nil {
				step(written, total)
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return er
		}
	}
	return nil
}

func uploadBytes(client *sftp.Client, data []byte, remotePath string) error {
	dst, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = dst.Write(data)
	return err
}

func formatSize(n int64) string {
	if n >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	if n >= 1024 {
		return fmt.Sprintf("%.0f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%d B", n)
}

func remoteVersion(client *ssh.Client) (string, error) {
	home, err := remoteHome(client)
	if err != nil {
		return "", err
	}
	bin := remoteAppBin(home)
	if remoteExecutable(client, bin) {
		out, err := runSession(client, shellQuote(bin)+" --version 2>/dev/null")
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out), nil
		}
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", err
	}
	defer sftpClient.Close()
	verFile := filepath.ToSlash(filepath.Join(home, remoteAppDir, "VERSION"))
	if v := strings.TrimSpace(readRemoteText(sftpClient, verFile)); v != "" {
		return v, nil
	}
	return "", nil
}

func uploadFile(client *sftp.Client, localPath, remotePath string) error {
	return uploadFileFast(client, localPath, remotePath)
}

func uploadDir(client *sftp.Client, localDir, remoteDir string) error {
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
		return uploadFile(client, path, rpath)
	})
}
