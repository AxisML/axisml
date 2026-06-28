#!/usr/bin/env bash
#
# Generate localized (en / zh-CN) screenshots of the Platform frontend.
#
# Boots the Vite SPA in its in-browser mock mode (no backend, no cluster), walks
# every route in each language with a headless Chromium, and writes the PNGs to
# docs/screenshots/en and docs/screenshots/zh-CN.
#
# This wrapper makes sure the dependencies are present, then hands off to
# scripts/gen_screenshots.py (Playwright for Python) which does the real work.
#
# Usage:
#   scripts/gen-screenshots.sh
#
# Requires: python3 + pnpm on PATH (pnpm is already this repo's frontend toolchain).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/axisml-platform/frontend"
VENV_DIR="$REPO_ROOT/.venv"

command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required on PATH" >&2; exit 1; }
command -v pnpm >/dev/null 2>&1 || { echo "error: pnpm is required on PATH" >&2; exit 1; }

# 1. Frontend deps (the dev server + mock API live here).
if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
  echo "[screenshots] installing frontend dependencies…"
  (cd "$FRONTEND_DIR" && pnpm install --frozen-lockfile)
fi

# 2. Playwright for Python, isolated in the project-root .venv so it never touches
#    the system Python or the frontend's dependency tree.
if [ ! -x "$VENV_DIR/bin/python" ]; then
  echo "[screenshots] creating Python venv + installing Playwright (one-time)…"
  python3 -m venv "$VENV_DIR"
  "$VENV_DIR/bin/pip" install --quiet --upgrade pip
  "$VENV_DIR/bin/pip" install --quiet playwright
fi

echo "[screenshots] ensuring Chromium is available…"
"$VENV_DIR/bin/python" -m playwright install chromium

# 3. Drive the captures.
"$VENV_DIR/bin/python" "$SCRIPT_DIR/gen_screenshots.py" "$@"
