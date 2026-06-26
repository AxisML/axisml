#!/usr/bin/env bash
# Assemble docs/configuration.md from the static preamble plus each service's
# generated reference section (printed by its cmd/config-doc-gen, which reflects
# over its Config struct). Usage: gen-config-doc.sh [output-path]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/docs/configuration.md}"
PREAMBLE="$ROOT/docs/configuration.preamble.md"

# axisml-core (Lite) is intentionally excluded — it is env-only and configured
# via its Docker Compose environment block, not this reference manual.
SERVICES=(
  "axisml-system/compute-service"
  "axisml-system/artifact-hub"
  "axisml-platform/backend"
)

tmp="$(mktemp)"
cat "$PREAMBLE" > "$tmp"
printf '\n# Per-service reference\n\n' >> "$tmp"
for svc in "${SERVICES[@]}"; do
  ( cd "$ROOT/$svc" && go run ./cmd/config-doc-gen ) >> "$tmp"
done

# Normalize EOF to a single trailing newline. Each rendered section ends with a
# blank line so sections are spaced apart; that leaves the final section with a
# trailing blank line, which the end-of-file-fixer pre-commit hook would strip —
# do it here so the generated doc and the hook agree (config-docs-test guard).
printf '%s\n' "$(<"$tmp")" > "$OUT"
rm -f "$tmp"
echo "wrote $OUT"
