#!/bin/sh
set -eu
ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== GoScan install-doctor ==="
echo ""

if [ -x "$ROOT/bin/goscan" ]; then
  echo "Build local:  $ROOT/bin/goscan"
else
  echo "Build local:  (não compilado — make release)"
fi

PREFIX="${PREFIX:-$HOME/.local}"
APP="$PREFIX/share/goscan/app"
DATA="$PREFIX/share/goscan/data"
SETTINGS="${XDG_CONFIG_HOME:-$HOME/.config}/goscan/settings.yaml"

if [ -f "$APP/.goscan-install" ]; then
  echo "Instalado:    sim ($APP)"
  if [ -f "$APP/VERSION" ]; then
    echo "Versão:       $(tr -d '\n' < "$APP/VERSION")"
  fi
else
  echo "Instalado:    não (make install)"
fi

echo ""
echo "--- settings ---"
if [ -f "$SETTINGS" ]; then
  echo "Ficheiro:     $SETTINGS"
  cat "$SETTINGS"
  if grep -q "$ROOT" "$SETTINGS" 2>/dev/null; then
    echo "⚠ settings apontam para o repo de dev — correr: make migrate-prod-data"
  fi
else
  echo "(sem settings — make install ou migrate-prod-data)"
fi

echo ""
echo "--- paths prod ---"
GOSCAN="$PREFIX/bin/goscan"
[ -x "$GOSCAN" ] || GOSCAN="$ROOT/bin/goscan"
if [ -x "$GOSCAN" ]; then
  env GOSCAN_MODE=prod "$GOSCAN" findings list --limit 3 2>/dev/null || echo "(erro ou vazio)"
else
  echo "(bin/goscan em falta)"
fi

echo ""
echo "--- dev ---"
echo "Repo:         $ROOT"
echo "dominios.db:  $(test -f "$ROOT/dominios.db" && du -sh "$ROOT/dominios.db" | cut -f1 || echo '—')"
echo "Dev UI:       make dev-ui  (GOSCAN_REPO_ROOT → repo)"
echo ""
echo "Prod dados:   $DATA"
echo "Prod scan:    $DATA/files"
