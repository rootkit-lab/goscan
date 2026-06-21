#!/bin/sh
# Publica goscan-remote num repositório git privado (releases de workers).
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARCH=linux-amd64
RELEASE_DIR="${WORKER_RELEASE_DIR:-$ROOT/dist/worker-release}"
REMOTE="${WORKER_RELEASE_REMOTE:-}"

VERSION="$(tr -d '\n' < "$ROOT/assets/VERSION" 2>/dev/null || echo 0.0.0-dev)"
TAG="v${VERSION#v}"

die() { echo "publish-worker: $*" >&2; exit 1; }

[ -n "$REMOTE" ] || die "Define WORKER_RELEASE_REMOTE (ex.: git@github.com:USER/goscan-worker-releases.git)"

echo "=== Publicar worker release $TAG ==="
make -C "$ROOT" build-remote

mkdir -p "$RELEASE_DIR/$ARCH"
cp "$ROOT/bin/goscan-remote" "$RELEASE_DIR/$ARCH/goscan-remote"
echo "$VERSION" > "$RELEASE_DIR/$ARCH/VERSION"
sha256sum "$RELEASE_DIR/$ARCH/goscan-remote" | awk '{print $1}' > "$RELEASE_DIR/$ARCH/BINHASH"
chmod +x "$ROOT/deploy/worker-release/install-worker.sh"
cp "$ROOT/deploy/worker-release/install-worker.sh" "$RELEASE_DIR/install-worker.sh"
cp "$ROOT/deploy/worker-release/README.md" "$RELEASE_DIR/README.md"

if [ ! -d "$RELEASE_DIR/.git" ]; then
  echo "A clonar $REMOTE → $RELEASE_DIR"
  git clone "$REMOTE" "$RELEASE_DIR"
fi

cd "$RELEASE_DIR"
git add -A
if git diff --cached --quiet; then
  echo "Sem alterações nos artefactos."
else
  git commit -m "release $TAG"
fi
git tag -f "$TAG"
BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo main)"
echo "A enviar branch $BRANCH e tag $TAG…"
git push origin "$BRANCH"
git push -f origin "$TAG"
echo "OK — $TAG publicado em $REMOTE"
echo "Hash: $(cat "$ARCH/BINHASH")"
