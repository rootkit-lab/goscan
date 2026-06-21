package remoteworker

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/crypto/ssh"
)

const remoteWorkerReleaseCache = ".local/share/goscan/worker-release"

// install-worker.sh runs on the remote host (git must be installed).
const workerInstallScript = `set -e
ARCH=linux-amd64
APP_DIR="$HOME/.local/share/goscan/app"
CACHE="$HOME/.local/share/goscan/worker-release"
REF="$GOSCAN_WORKER_REF"
REPO="$GOSCAN_WORKER_REPO"

if ! command -v git >/dev/null 2>&1; then
  echo "git em falta no filho — instala git ou usa deploy SFTP" >&2
  exit 1
fi
if [ -z "$REPO" ] || [ -z "$REF" ]; then
  echo "GOSCAN_WORKER_REPO/REF em falta" >&2
  exit 1
fi

mkdir -p "$(dirname "$CACHE")"
if [ ! -d "$CACHE/.git" ]; then
  git clone --depth 1 --branch "$REF" "$REPO" "$CACHE"
else
  cd "$CACHE"
  git fetch --depth 1 origin "$REF"
  git checkout -f FETCH_HEAD
fi

BIN="$CACHE/$ARCH/goscan-remote"
if [ ! -f "$BIN" ]; then
  echo "binário em falta: $BIN" >&2
  exit 1
fi

mkdir -p "$APP_DIR/bin" "$HOME/.local/bin"
install -m 755 "$BIN" "$APP_DIR/bin/goscan-remote"
if [ -f "$CACHE/$ARCH/VERSION" ]; then cp "$CACHE/$ARCH/VERSION" "$APP_DIR/VERSION"; fi
if [ -f "$CACHE/$ARCH/BINHASH" ]; then cp "$CACHE/$ARCH/BINHASH" "$APP_DIR/BINHASH"; fi
ln -sf "$APP_DIR/bin/goscan-remote" "$HOME/.local/bin/goscan-remote"
echo GOSCAN_WORKER_INSTALL_OK
`

func installFromGit(client *ssh.Client, w Config, home, localVer, localHash string, onLog func(string)) error {
	repo := w.DeployRepo.Normalized()
	ref := repo.Ref
	if ref == "" {
		ref = "v" + strings.TrimPrefix(strings.TrimSpace(localVer), "v")
	}
	cloneURL := gitCloneURL(repo.URL, repo.Token)
	onLog(fmt.Sprintf("a actualizar via git (%s · %s)…", shortRepoLabel(repo.URL), ref))
	script := fmt.Sprintf(
		`export GOSCAN_WORKER_REPO=%s GOSCAN_WORKER_REF=%s HOME=%s
%s`,
		shellQuote(cloneURL), shellQuote(ref), shellQuote(home),
		workerInstallScript,
	)
	out, err := runSession(client, script)
	if err != nil {
		return fmt.Errorf("install git: %w", err)
	}
	if !strings.Contains(out, "GOSCAN_WORKER_INSTALL_OK") {
		return fmt.Errorf("install git: resposta inesperada")
	}
	return nil
}

func gitCloneURL(raw, token string) string {
	raw = strings.TrimSpace(raw)
	if token == "" || strings.HasPrefix(raw, "git@") || strings.HasPrefix(raw, "ssh://") {
		return raw
	}
	if !strings.HasPrefix(raw, "https://") {
		return raw
	}
	rest := strings.TrimPrefix(raw, "https://")
	// GitHub/GitLab: https://x-access-token:TOKEN@host/path
	return "https://x-access-token:" + url.QueryEscape(token) + "@" + rest
}

func shortRepoLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "repo"
	}
	if i := strings.LastIndex(raw, "/"); i >= 0 && i < len(raw)-1 {
		s := raw[i+1:]
		return strings.TrimSuffix(s, ".git")
	}
	return raw
}
