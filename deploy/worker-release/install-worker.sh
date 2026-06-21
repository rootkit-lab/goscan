#!/bin/sh
# Instala goscan-remote a partir deste repositório (corre no VPS).
set -e
ARCH=linux-amd64
APP_DIR="${HOME}/.local/share/goscan/app"
ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN="$ROOT/$ARCH/goscan-remote"

[ -f "$BIN" ] || { echo "binário em falta: $BIN" >&2; exit 1; }

mkdir -p "$APP_DIR/bin" "${HOME}/.local/bin"
install -m 755 "$BIN" "$APP_DIR/bin/goscan"
[ -f "$ROOT/$ARCH/VERSION" ] && cp "$ROOT/$ARCH/VERSION" "$APP_DIR/VERSION"
[ -f "$ROOT/$ARCH/BINHASH" ] && cp "$ROOT/$ARCH/BINHASH" "$APP_DIR/BINHASH"
ln -sf "$APP_DIR/bin/goscan" "${HOME}/.local/bin/goscan"
echo "goscan worker instalado em $APP_DIR/bin/goscan"
