#!/usr/bin/env bash
set -euo pipefail

# Merge per-component Go coverage profiles into a single file.
#
# Usage: merge-coverage.sh <output> <component-dir> [<component-dir>...]
#
# Each component is expected to produce coverage.out and/or envtest-coverage.out
# under <component-dir>/coverage/. Profiles must use -covermode=atomic so the
# merged file's mode header stays consistent.

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <output> <component-dir> [<component-dir>...]" >&2
  exit 2
fi

out="$1"; shift
mkdir -p "$(dirname "$out")"

# A partial merge would leave a half-written profile that downstream tools
# would happily consume — drop it on any error so the next run starts clean.
trap 'rm -f "$out"' ERR

echo "mode: atomic" > "$out"

merged=0
for dir in "$@"; do
  for f in "$dir/coverage/coverage.out" "$dir/coverage/envtest-coverage.out"; do
    if [ -f "$f" ]; then
      tail -n +2 "$f" >> "$out"
      merged=$((merged + 1))
    fi
  done
done

if [ "$merged" -eq 0 ]; then
  echo "WARN: no per-component coverage profiles found; merged file is empty." >&2
  echo "      Run 'make coverage-unit' and/or 'make coverage-envtest' first." >&2
  exit 0
fi

echo ">>> merged $merged profile(s) into $out"

# `go tool cover -func` resolves package paths against the current module, so
# it can't summarize a merged profile from the repo root (no root go.mod). The
# merged file is still valid — Codecov and `go tool cover -html` parse it fine
# from a Go-module context. For a local CLI summary, run cover -func from each
# component dir (`go test` already prints per-package coverage during the run).
profile_blocks=$(($(wc -l < "$out") - 1))
echo ">>> $profile_blocks profile blocks recorded"
