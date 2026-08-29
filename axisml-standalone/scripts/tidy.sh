#!/usr/bin/env bash
# Tidy the publishable module through temporary local replacements. The
# committed go.mod stays free of replace directives, while the monorepo remains
# buildable before matching System module tags are published.
set -euo pipefail

standalone_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
system_root="$(cd "$standalone_root/../axisml-system" && pwd)"
tidy_dir="$(mktemp -d)"
trap 'rm -rf "$tidy_dir"' EXIT

system_modules=(
  apis
  artifact-hub
  cluster-manager
  compute-service
)

cp "$standalone_root/go.mod" "$tidy_dir/standalone.mod"
cp "$standalone_root/go.sum" "$tidy_dir/standalone.sum"

for module in "${system_modules[@]}"; do
  GOWORK=off go mod edit -modfile="$tidy_dir/standalone.mod" \
    -replace="github.com/axisml/axisml/axisml-system/$module=$system_root/$module"
done
GOWORK=off go mod tidy -modfile="$tidy_dir/standalone.mod"
for module in "${system_modules[@]}"; do
  GOWORK=off go mod edit -modfile="$tidy_dir/standalone.mod" \
    -dropreplace="github.com/axisml/axisml/axisml-system/$module"
done

cp "$tidy_dir/standalone.mod" "$standalone_root/go.mod"
cp "$tidy_dir/standalone.sum" "$standalone_root/go.sum"
