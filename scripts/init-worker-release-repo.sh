#!/bin/sh
# Inicializa o clone local do repositório privado de worker releases.
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RELEASE_DIR="${WORKER_RELEASE_DIR:-$ROOT/dist/worker-release}"
REMOTE="${WORKER_RELEASE_REMOTE:-}"

die() { echo "init-worker-release-repo: $*" >&2; exit 1; }

[ -n "$REMOTE" ] || die "Define WORKER_RELEASE_REMOTE (ex.: git@github.com:USER/goscan-worker-releases.git)"

if [ -d "$RELEASE_DIR/.git" ]; then
  echo "Já existe: $RELEASE_DIR"
  exit 0
fi

mkdir -p "$(dirname "$RELEASE_DIR")"
echo "A clonar $REMOTE…"
git clone "$REMOTE" "$RELEASE_DIR" || {
  echo ""
  echo "Se o repositório remoto ainda não existe:"
  echo "  1. Cria repo privado vazio (GitHub/GitLab)"
  echo "  2. git init && git remote add origin $REMOTE && git push -u origin main"
  echo "  3. Volta a correr: make init-worker-release-repo"
  exit 1
}
echo "OK — clone em $RELEASE_DIR"
echo "Seguinte: make publish-worker"
