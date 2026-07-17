#!/usr/bin/env bash
#
# Publish the AxisML Go modules consumed by out-of-tree composition roots such
# as aios-ml. The external composition root and standalone runtime are owned by
# that project; this script publishes only the shared AxisML modules it consumes.
#
# Local development keeps v0.0.0 requirements plus relative replace directives.
# External consumers ignore dependency-module replaces, so a release must first
# bump every in-repo requirement to a real version, then tag each module using
# Go's subdirectory tag convention.
#
# Usage:
#   scripts/publish-modules.sh list
#   scripts/publish-modules.sh bump v0.1.0
#   scripts/publish-modules.sh tag v0.1.0 [--push]

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Runtime modules used by aios-ml plus the shared OpenAPI builder used by its
# control-plane document generator. pkg/configdoc remains in-repo-only because
# no package in the external build graph imports it.
MODULES=(
  axisml-system/apis
  axisml-system/artifact-hub
  axisml-system/cluster-manager
  axisml-system/compute-operator
  axisml-system/compute-service
  axisml-system/tenant-operator
  pkg/axismlconfig
  pkg/openapigen
)

MODPREFIX="github.com/axisml/axisml"

is_publishable() {
  local path="$1" dir
  for dir in "${MODULES[@]}"; do
    [ "$path" = "$MODPREFIX/$dir" ] && return 0
  done
  return 1
}

in_repo_requires() {
  grep "$MODPREFIX/" "$1" \
    | grep -v '=>' \
    | grep -vE '^[[:space:]]*(module|replace)[[:space:]]' \
    | awk -v prefix="$MODPREFIX/" '{for (i=1;i<=NF;i++) if (index($i,prefix)==1) print $i}' \
    | sort -u
}

validate_version() {
  case "$1" in
    v[0-9]*.[0-9]*.[0-9]*) ;;
    *) echo "error: version must look like vMAJOR.MINOR.PATCH, got: $1" >&2; exit 2 ;;
  esac
}

cmd_list() {
  local dir
  for dir in "${MODULES[@]}"; do
    echo "$MODPREFIX/$dir"
  done
}

cmd_bump() {
  local version="$1" dir modfile path bumped
  validate_version "$version"

  for dir in "${MODULES[@]}"; do
    modfile="$dir/go.mod"
    [ -f "$modfile" ] || { echo "error: missing $modfile" >&2; exit 1; }
    bumped=0
    while IFS= read -r path; do
      [ -n "$path" ] || continue
      if is_publishable "$path"; then
        (cd "$dir" && go mod edit -require="$path@$version") || exit 1
        bumped=1
      fi
    done < <(in_repo_requires "$modfile")

    if [ "$bumped" -eq 1 ]; then
      echo ">>> $dir: bumped in-repo requirements to $version"
      (cd "$dir" && go mod tidy) || exit 1
    else
      echo ">>> $dir: no published in-repo requirements"
    fi
  done

  echo "Review and commit the module changes, then run: $0 tag $version"
}

cmd_tag() {
  local version="$1" push=0 dir tag
  local -a created=()
  validate_version "$version"
  [ "${2:-}" = "--push" ] && push=1

  if [ -n "$(git status --porcelain)" ]; then
    echo "error: working tree is dirty; commit the version bump first" >&2
    exit 1
  fi

  for dir in "${MODULES[@]}"; do
    tag="$dir/$version"
    if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
      echo ">>> tag exists, skipping: $tag"
      continue
    fi
    git tag -a "$tag" -m "$dir $version" || exit 1
    created+=("$tag")
    echo ">>> created: $tag"
  done

  [ "${#created[@]}" -gt 0 ] || { echo "No new tags created."; return 0; }
  if [ "$push" -eq 1 ]; then
    git push origin "${created[@]}"
  else
    echo "Push with: git push origin ${created[*]}"
  fi
}

case "${1:-}" in
  list) cmd_list ;;
  bump) shift; [ $# -ge 1 ] || { echo "usage: $0 bump <version>" >&2; exit 2; }; cmd_bump "$1" ;;
  tag) shift; [ $# -ge 1 ] || { echo "usage: $0 tag <version> [--push]" >&2; exit 2; }; cmd_tag "$@" ;;
  *) echo "usage: $0 {list|bump <version>|tag <version> [--push]}" >&2; exit 2 ;;
esac
