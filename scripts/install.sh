#!/bin/sh
# Instala GoScan em PREFIX (default ~/.local) — app read-only + data separado.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
APP_DIR="$PREFIX/share/goscan/app"
DATA_DIR="$PREFIX/share/goscan/data"
BIN_DIR="$PREFIX/bin"
DESKTOP_DIR="$PREFIX/share/applications"
ICON_DIR="$PREFIX/share/icons/hicolor/256x256/apps"

die() { echo "install: $*" >&2; exit 1; }

[ -x "$ROOT/bin/goscan" ] || die "bin/goscan em falta — corre make release primeiro"
[ -x "$ROOT/bin/goscan-ui" ] || die "bin/goscan-ui em falta — corre make release primeiro"
[ -f "$ROOT/scripts/registry.yaml" ] || die "scripts/registry.yaml em falta"

# Ícone PNG
if [ ! -f "$ROOT/assets/icon/goscan.png" ]; then
  python3 "$ROOT/scripts/icon-to-png.py" || die "falha a gerar ícone PNG"
fi

echo "Instalar GoScan → PREFIX=$PREFIX"

mkdir -p "$APP_DIR/bin" "$APP_DIR/scripts" "$DATA_DIR/var/findings" "$DATA_DIR/var/logs/batch" "$BIN_DIR" "$DESKTOP_DIR" "$ICON_DIR"

install -m 755 "$ROOT/bin/goscan" "$APP_DIR/bin/goscan"
install -m 755 "$ROOT/bin/goscan-ui" "$APP_DIR/bin/goscan-ui"

# Scripts (sem .venv do dev)
rsync -a --delete \
  --exclude '.venv' \
  --exclude '__pycache__' \
  --exclude '*.pyc' \
  "$ROOT/scripts/" "$APP_DIR/scripts/"

if [ -f "$ROOT/assets/VERSION" ]; then
  install -m 644 "$ROOT/assets/VERSION" "$APP_DIR/VERSION"
fi
touch "$APP_DIR/.goscan-install"

# Venv prod
VENV="$APP_DIR/scripts/.venv"
if [ ! -x "$VENV/bin/python" ]; then
  echo "A criar venv Python em app/scripts/.venv…"
  python3 -m venv "$VENV" || die "python3-venv em falta (sudo apt install python3-venv python3-full)"
fi
echo "A instalar dependências Python…"
"$VENV/bin/pip" install -q -U pip
"$VENV/bin/pip" install -q -r "$APP_DIR/scripts/requirements.txt"

ln -sf "$APP_DIR/bin/goscan" "$BIN_DIR/goscan"
ln -sf "$APP_DIR/bin/goscan-ui" "$BIN_DIR/goscan-ui"

install -m 644 "$ROOT/assets/icon/goscan.png" "$ICON_DIR/goscan.png"

DESKTOP="$DESKTOP_DIR/goscan-ui.desktop"
cat > "$DESKTOP" <<EOF
[Desktop Entry]
Type=Application
Name=GoScan
Comment=Findings, checkers e scan de .env
Exec=$APP_DIR/bin/goscan-ui
Icon=goscan
Terminal=false
Categories=Development;Security;
StartupWMClass=goscan-ui
TryExec=$APP_DIR/bin/goscan-ui
EOF

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
fi

# Settings prod (XDG) — não apontar para repo de dev
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/goscan"
SETTINGS="$CONFIG_DIR/settings.yaml"
SCAN_DIR="$DATA_DIR/files"
mkdir -p "$CONFIG_DIR" "$SCAN_DIR"
if [ ! -f "$SETTINGS" ] || [ "${FORCE_SETTINGS:-0}" = "1" ]; then
  cat > "$SETTINGS" <<EOF
data_dir: $DATA_DIR
scan_dir: $SCAN_DIR
EOF
  echo "Settings prod → $SETTINGS"
fi

echo ""
echo "✓ GoScan instalado"
echo "  CLI:      $BIN_DIR/goscan"
echo "  UI:       $BIN_DIR/goscan-ui  (ou menu GoScan)"
echo "  App:      $APP_DIR"
echo "  Dados:    $DATA_DIR"
echo "  Dev UI:   make dev-ui  (dados no repo — separado)"
echo ""
echo "Verificar: make install-doctor"
