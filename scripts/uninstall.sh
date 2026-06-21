#!/bin/sh
set -eu
PREFIX="${PREFIX:-$HOME/.local}"
APP_DIR="$PREFIX/share/goscan/app"
DATA_DIR="$PREFIX/share/goscan/data"
BIN_DIR="$PREFIX/bin"
DESKTOP="$PREFIX/share/applications/goscan-ui.desktop"
ICON="$PREFIX/share/icons/hicolor/256x256/apps/goscan.png"

rm -f "$BIN_DIR/goscan" "$BIN_DIR/goscan-ui" "$DESKTOP" "$ICON"
rm -rf "$APP_DIR"
echo "Removido app de $PREFIX (dados mantidos em $DATA_DIR)"
