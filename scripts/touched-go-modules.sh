#!/usr/bin/env bash
#
# touched-go-modules.sh — run a per-module action against only the Go modules
# that contain the file paths passed on the command line.
#
# Usage:
#   touched-go-modules.sh <action> [file ...]
#
# Actions:
#   vet            go vet -tags=integration ./...
#   golangci-lint  golangci-lint run --build-tags=integration
#   test           go test -short -tags=integration ./...
#
# Designed for the pre-commit framework, which passes matched files as args.
# We resolve each file to the nearest enclosing go.mod, dedupe, then run the
# action once per module. Walking just the touched modules keeps the hook
# fast: a typical commit touches one module; CI walks all twelve.
#
# Exits non-zero on the first action failure but lets the failing tool's
# output speak for itself (no extra wrapping).

set -uo pipefail

# pre-commit's `language: system` hooks inherit the user's PATH but not the
# common Go-install locations (most install golangci-lint via `go install`
# into $GOPATH/bin). Extend PATH so the hook works regardless of how the
# user set up their shell rc.
if command -v go >/dev/null 2>&1; then
  PATH="$(go env GOPATH)/bin:$PATH"
  export PATH
fi

if [ $# -lt 1 ]; then
  echo "usage: $0 <vet|golangci-lint|test> [file ...]" >&2
  exit 2
fi

action="$1"; shift

# Sanity-check that the required tool is reachable before walking modules,
# so the error surfaces once at the top instead of N times per module.
case "$action" in
  vet|test)
    command -v go >/dev/null || { echo "go not on PATH"; exit 2; }
    ;;
  golangci-lint)
    if ! command -v golangci-lint >/dev/null; then
      cat >&2 <<'MSG'
golangci-lint not found on PATH (or in $(go env GOPATH)/bin).
Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.1
MSG
      exit 2
    fi
    ;;
esac

# When no files were passed (e.g. user ran pre-commit against a non-go set)
# nothing is touched — exit 0 silently rather than misleading the caller.
if [ $# -eq 0 ]; then
  exit 0
fi

# Resolve each file to its containing module by walking up to the nearest
# go.mod. Files outside any module are skipped silently.
resolve_module() {
  local dir; dir=$(dirname "$1")
  while [ "$dir" != "." ] && [ "$dir" != "/" ]; do
    if [ -f "$dir/go.mod" ]; then
      printf '%s\n' "$dir"
      return
    fi
    dir=$(dirname "$dir")
  done
}

modules=$(
  for f in "$@"; do
    resolve_module "$f"
  done | sort -u
)

if [ -z "$modules" ]; then
  exit 0
fi

status=0
while IFS= read -r mod; do
  printf '>>> %s (%s)\n' "$mod" "$action"
  case "$action" in
    vet)
      (cd "$mod" && go vet -tags=integration ./...) || status=1
      ;;
    golangci-lint)
      (cd "$mod" && golangci-lint run --build-tags=integration) || status=1
      ;;
    test)
      # Unit tests only (no -tags=integration); integration tests need
      # envtest + testcontainers which we don't want firing on every push.
      # -short lets tests that opt in skip themselves; no -race here since
      # race is a CI concern and would roughly double the runtime.
      (cd "$mod" && go test -short ./...) || status=1
      ;;
    *)
      echo "unknown action: $action" >&2
      exit 2
      ;;
  esac
done <<<"$modules"

exit "$status"
