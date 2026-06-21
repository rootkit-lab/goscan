#!/bin/sh
# Preenche o layout de app prod (igual install.sh, destino configurável).
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
DEST="${1:?destino app dir (ex.: dist/stage/usr/local/share/goscan/app)}"

cd "$ROOT"

if [ "${SKIP_RELEASE:-0}" != "1" ]; then
  make release
fi

mkdir -p "$DEST/bin" "$DEST/scripts"

[ -x "$ROOT/bin/goscan" ] || { echo "stage: bin/goscan em falta — make release" >&2; exit 1; }
[ -x "$ROOT/bin/goscan-ui" ] || { echo "stage: bin/goscan-ui em falta" >&2; exit 1; }
[ -x "$ROOT/bin/goscan-remote" ] || { echo "stage: bin/goscan-remote em falta" >&2; exit 1; }

install -m 755 "$ROOT/bin/goscan" "$ROOT/bin/goscan-ui" "$ROOT/bin/goscan-remote" "$DEST/bin/"

rsync -a --delete \
  --exclude '.venv' \
  --exclude '__pycache__' \
  --exclude '*.pyc' \
  "$ROOT/scripts/" "$DEST/scripts/"

if [ -f "$ROOT/assets/VERSION" ]; then
  install -m 644 "$ROOT/assets/VERSION" "$DEST/VERSION"
fi
touch "$DEST/.goscan-install"

if [ ! -f "$ROOT/assets/icon/goscan.png" ]; then
  python3 "$ROOT/scripts/icon-to-png.py"
fi

echo "stage: $DEST ($(tr -d '\n' < "$ROOT/assets/VERSION" 2>/dev/null || echo dev))"
