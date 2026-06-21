#!/bin/sh
# Bump VERSION, commit, tag vX.Y.Z e push — dispara GitHub Actions release.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(tr -d '\n\r' < "$ROOT/assets/VERSION" 2>/dev/null || true)"
fi
VERSION="${VERSION#v}"
[ -n "$VERSION" ] || { echo "release-publish: versão em falta (ex.: 1.0.1)" >&2; exit 1; }

TAG="v${VERSION}"
echo "$VERSION" > "$ROOT/assets/VERSION"

echo "=== Publicar release $TAG ==="

git add "$ROOT/assets/VERSION"
if git diff --cached --quiet; then
  echo "assets/VERSION já em $VERSION"
else
  git commit -m "Release $TAG"
fi

if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Tag $TAG já existe localmente — a mover para HEAD"
  git tag -f -a "$TAG" -m "Release $TAG"
else
  git tag -a "$TAG" -m "Release $TAG"
fi

echo "A enviar main + tag (GitHub Actions gera .deb e .msi)…"
git push origin HEAD
git push origin "$TAG" --force

echo ""
echo "✓ Tag $TAG enviada"
echo "  Acompanhar: https://github.com/rootkit-lab/goscan/actions"
echo "  Instalar local: make release && make install"
