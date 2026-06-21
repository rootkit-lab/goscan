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

install_bin() {
  name="$1"
  src="$ROOT/bin/$name"
  if [ ! -x "$src" ]; then
    src="$ROOT/bin/${name}.exe"
  fi
  [ -x "$src" ] || { echo "stage: bin/$name em falta — make release" >&2; exit 1; }
  dest="$DEST/bin/$name"
  case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) dest="$DEST/bin/${name}.exe" ;;
  esac
  install -m 755 "$src" "$dest"
}

install_bin goscan
install_bin goscan-ui
install_bin goscan-remote

if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete \
    --exclude '.venv' \
    --exclude '__pycache__' \
    --exclude '*.pyc' \
    "$ROOT/scripts/" "$DEST/scripts/"
else
  rm -rf "$DEST/scripts"
  mkdir -p "$DEST/scripts"
  cp -R "$ROOT/scripts/." "$DEST/scripts/"
  rm -rf "$DEST/scripts/.venv" "$DEST/scripts/__pycache__"
  find "$DEST/scripts" -name '*.pyc' -delete 2>/dev/null || true
fi

if [ -f "$ROOT/assets/VERSION" ]; then
  install -m 644 "$ROOT/assets/VERSION" "$DEST/VERSION"
fi
touch "$DEST/.goscan-install"

if [ ! -f "$ROOT/assets/icon/goscan.png" ]; then
  python3 "$ROOT/scripts/icon-to-png.py"
fi

echo "stage: $DEST ($(tr -d '\n' < "$ROOT/assets/VERSION" 2>/dev/null || echo dev))"
