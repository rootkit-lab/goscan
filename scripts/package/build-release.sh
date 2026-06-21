#!/bin/sh
# Build prod (icon + binários). Venv Python só em Unix — Windows/deb postinst tratam checkers.
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 scripts/icon-to-png.py

case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*)
    ;;
  *)
    make scripts-venv
    ;;
esac

make build-ui build

echo "build-release: OK ($(tr -d '\n\r' < assets/VERSION 2>/dev/null || echo dev))"
