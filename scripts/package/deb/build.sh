#!/bin/sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/../../.." && pwd)"
VERSION="$(tr -d '\n\r' < "$ROOT/assets/VERSION" 2>/dev/null || echo 0.0.0-dev)"
PKGROOT="$ROOT/dist/package/deb/root"
OUT="$ROOT/dist/goscan_${VERSION}_amd64.deb"

rm -rf "$ROOT/dist/package/deb"
mkdir -p "$PKGROOT/usr/local/bin" \
  "$PKGROOT/usr/local/share/applications" \
  "$PKGROOT/usr/local/share/icons/hicolor/256x256/apps" \
  "$PKGROOT/DEBIAN"

SKIP_RELEASE=1 "$ROOT/scripts/package/stage.sh" "$PKGROOT/usr/local/share/goscan/app"

ln -sf ../share/goscan/app/bin/goscan "$PKGROOT/usr/local/bin/goscan"
ln -sf ../share/goscan/app/bin/goscan-ui "$PKGROOT/usr/local/bin/goscan-ui"
ln -sf ../share/goscan/app/bin/goscan-remote "$PKGROOT/usr/local/bin/goscan-remote"

install -m 644 "$ROOT/assets/icon/goscan.png" \
  "$PKGROOT/usr/local/share/icons/hicolor/256x256/apps/goscan.png"

cat > "$PKGROOT/usr/local/share/applications/goscan-ui.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=GoScan
Comment=Findings, checkers e scan de .env
Exec=/usr/local/share/goscan/app/bin/goscan-ui
Icon=goscan
Terminal=false
Categories=Development;Security;
StartupWMClass=goscan-ui
TryExec=/usr/local/share/goscan/app/bin/goscan-ui
EOF

cat > "$PKGROOT/DEBIAN/control" <<EOF
Package: goscan
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Rafael <me@rafaelroot.com>
Depends: python3, python3-venv, python3-pip
Homepage: https://github.com/rootkit-lab/goscan
Description: Scanner HTTP de ficheiros .env expostos
 GoScan — CLI, UI desktop (Wails) e checkers Python.
EOF

install -m 755 "$ROOT/scripts/package/deb/postinst" "$PKGROOT/DEBIAN/postinst"

mkdir -p "$ROOT/dist"
dpkg-deb --root-owner-group --build "$PKGROOT" "$OUT"
echo "✓ $OUT"
