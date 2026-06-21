#!/bin/sh
# Copia dados do repo de dev para directórios prod XDG e actualiza settings.yaml.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
DATA_DIR="$PREFIX/share/goscan/data"
SCAN_DIR="$DATA_DIR/files"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/goscan"
SETTINGS="$CONFIG_DIR/settings.yaml"

die() { echo "migrate-prod-data: $*" >&2; exit 1; }

size_of() {
  if [ -e "$1" ]; then
    du -sh "$1" 2>/dev/null | cut -f1
  else
    echo "—"
  fi
}

echo "=== GoScan migrate dev → prod ==="
echo "Origem (dev):  $ROOT"
echo "Destino (prod): $DATA_DIR"
echo ""

mkdir -p "$DATA_DIR/var/findings/by-domain" "$DATA_DIR/var/logs/batch" "$SCAN_DIR"

if [ -f "$ROOT/dominios.db" ]; then
  echo "→ dominios.db ($(size_of "$ROOT/dominios.db"))"
  cp -a "$ROOT/dominios.db" "$DATA_DIR/dominios.db"
else
  echo "→ dominios.db (não existe no repo — ignorado)"
fi

if [ -d "$ROOT/var/findings" ]; then
  echo "→ var/findings/ ($(size_of "$ROOT/var/findings"))"
  rsync -a "$ROOT/var/findings/" "$DATA_DIR/var/findings/"
else
  echo "→ var/findings/ (não existe — ignorado)"
fi

if [ -d "$ROOT/var/logs/batch" ]; then
  echo "→ var/logs/batch/ ($(size_of "$ROOT/var/logs/batch"))"
  rsync -a "$ROOT/var/logs/batch/" "$DATA_DIR/var/logs/batch/"
fi

if [ -d "$ROOT/files" ]; then
  echo "→ files/ → $SCAN_DIR ($(size_of "$ROOT/files"))"
  rsync -a "$ROOT/files/" "$SCAN_DIR/"
else
  echo "→ files/ (não existe — ignorado)"
fi

mkdir -p "$CONFIG_DIR"
cat > "$SETTINGS" <<EOF
data_dir: $DATA_DIR
scan_dir: $SCAN_DIR
EOF

echo ""
echo "Settings: $SETTINGS"
echo "  data_dir: $DATA_DIR"
echo "  scan_dir: $SCAN_DIR"
echo ""

GOSCAN="${PREFIX}/bin/goscan"
if [ ! -x "$GOSCAN" ]; then
  GOSCAN="$ROOT/bin/goscan"
fi

if [ -x "$GOSCAN" ]; then
  echo "--- findings prod (amostra) ---"
  env GOSCAN_MODE=prod "$GOSCAN" findings list --limit 5 || true
else
  echo "(compila bin/goscan ou make install para validar findings)"
fi

echo ""
echo "✓ Migração concluída — repo de dev intacto."
echo "  Prod:  $DATA_DIR"
echo "  Dev:   make dev-ui (repo $ROOT)"
echo "  Seguinte: make release && make install"
