#!/usr/bin/env bash
#
# publish-modules.sh — release the axisml-core library and its in-repo Go module
# dependencies so an OUT-OF-TREE project can `go get` them without any `replace`
# directives.
#
# The repo is a pure multi-module tree: axisml-core and its 6 in-repo
# dependencies are wired for local development via per-module `replace ../...`
# directives and pinned at the placeholder `v0.0.0`. Go IGNORES replace
# directives in dependency modules, so an external consumer must resolve real,
# published versions. This script bumps the in-repo `require`s to a real version
# (keeping the replaces — Go uses them for local dev, external consumers ignore
# them) and creates Go's per-subdirectory version tags (`<subdir>/vX.Y.Z`).
#
# The doc-gen-only libraries pkg/openapigen and pkg/configdoc are deliberately
# NOT published: nothing in axisml-core's import graph reaches them (the two
# doc-gen commands that did live in the separate axisml-core/tools module now),
# so Go 1.17+ module-graph pruning drops them from axisml-core's build list.
# The System components still `require` them for their own in-tree doc-gen, but
# that require is pruned away for an axisml-core consumer, so it never needs a
# real version — leaving both at v0.0.0 + a local replace is correct.
#
# Usage:
#   publish-modules.sh list
#   publish-modules.sh bump <version>            # edit go.mods (leaves tree dirty for review)
#   publish-modules.sh tag  <version> [--push]   # tag the current commit (and optionally push)
#
# Release flow:
#   1. scripts/publish-modules.sh bump v0.1.0
#   2. review the go.mod changes, then `git commit` them
#   3. scripts/publish-modules.sh tag v0.1.0 --push
#
# An external consumer then needs no replace block:
#   go get github.com/axisml/axisml/axisml-lite/axisml-core@v0.1.0
# (Go maps that to the `axisml-lite/axisml-core/v0.1.0` tag.) If the GitHub repo
# is private, the consumer sets GOPRIVATE=github.com/axisml/* plus git auth.
#
# NOTE: module tags are `<subdir>/vX.Y.Z` and never collide with the repo-wide
# `vX.Y.Z` Docker-image tags in .github/workflows/ci.yml — separate namespaces.

set -uo pipefail

# Repo root = parent of this script's dir, regardless of CWD.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# The full transitive set of in-repo modules in axisml-core's library build
# graph. Excluded: test-only modules (testutil, */test/integration), the
# dev-only axisml-core/tools module, and the doc-gen-only pkg/openapigen +
# pkg/configdoc (pruned out of axisml-core's graph — see the header note).
# Each module's path is github.com/axisml/axisml/<dir>.
MODULES=(
  axisml-lite/axisml-core
  axisml-system/artifact-hub
  axisml-system/cluster-manager
  axisml-system/compute-operator
  axisml-system/compute-service
  axisml-system/tenant-operator
  pkg/axismlconfig
)

MODPREFIX="github.com/axisml/axisml"

# is_publishable <module-path> → 0 if the path is one of the 7 modules above.
is_publishable() {
  local p="$1" dir
  for dir in "${MODULES[@]}"; do
    [ "$p" = "$MODPREFIX/$dir" ] && return 0
  done
  return 1
}

# in_repo_requires <go.mod> → prints the in-repo require paths (not replaces).
in_repo_requires() {
  grep "$MODPREFIX/" "$1" \
    | grep -v '=>' \
    | grep -vE '^[[:space:]]*(module|replace)[[:space:]]' \
    | awk -v pfx="$MODPREFIX/" '{for (i=1;i<=NF;i++) if (index($i,pfx)==1) print $i}' \
    | sort -u
}

validate_version() {
  case "$1" in
    v[0-9]*.[0-9]*.[0-9]*) ;;
    *) echo "error: version must look like vMAJOR.MINOR.PATCH (e.g. v0.1.0), got: $1" >&2; exit 2 ;;
  esac
}

cmd_list() {
  for dir in "${MODULES[@]}"; do echo "$MODPREFIX/$dir"; done
}

cmd_bump() {
  local version="$1"; validate_version "$version"
  command -v go >/dev/null || { echo "go not on PATH" >&2; exit 2; }

  for dir in "${MODULES[@]}"; do
    local modfile="$dir/go.mod"
    [ -f "$modfile" ] || { echo "error: missing $modfile" >&2; exit 1; }
    local bumped=0 path
    while IFS= read -r path; do
      [ -n "$path" ] || continue
      if is_publishable "$path"; then
        (cd "$dir" && go mod edit -require="$path@$version") || exit 1
        bumped=1
      fi
    done < <(in_repo_requires "$modfile")
    if [ "$bumped" -eq 1 ]; then
      echo ">>> $dir: bumped in-repo requires → $version; tidying"
      (cd "$dir" && go mod tidy) || exit 1
    else
      echo ">>> $dir: no in-repo requires (leaf module)"
    fi
  done

  echo
  echo "Bumped in-repo requires to $version across ${#MODULES[@]} modules (replaces kept for local dev)."
  echo "Review the go.mod/go.sum changes, then commit them before running: $0 tag $version"
}

cmd_tag() {
  local version="$1"; validate_version "$version"
  local push=0
  [ "${2:-}" = "--push" ] && push=1

  # Tags point at a commit, not the working tree — require a clean tree so the
  # tagged commit actually contains the version bumps.
  if [ -n "$(git status --porcelain)" ]; then
    echo "error: working tree is dirty. Commit the 'bump $version' changes before tagging." >&2
    exit 1
  fi

  local created=()
  for dir in "${MODULES[@]}"; do
    local tag="$dir/$version"
    if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
      echo ">>> tag exists, skipping: $tag"
      continue
    fi
    git tag -a "$tag" -m "$dir $version" || exit 1
    created+=("$tag")
    echo ">>> created tag: $tag"
  done

  if [ "${#created[@]}" -eq 0 ]; then
    echo "No new tags created."
    return 0
  fi

  if [ "$push" -eq 1 ]; then
    echo ">>> pushing ${#created[@]} tags to origin"
    git push origin "${created[@]}" || exit 1
  else
    echo
    echo "Created ${#created[@]} local tags. Push them with:"
    echo "  git push origin ${created[*]}"
    echo "or re-run: $0 tag $version --push"
  fi
}

case "${1:-}" in
  list) cmd_list ;;
  bump) shift; [ $# -ge 1 ] || { echo "usage: $0 bump <version>" >&2; exit 2; }; cmd_bump "$1" ;;
  tag)  shift; [ $# -ge 1 ] || { echo "usage: $0 tag <version> [--push]" >&2; exit 2; }; cmd_tag "$@" ;;
  *)
    cat >&2 <<EOF
usage: $0 <command>

commands:
  list                     print the module paths that get published/tagged
  bump <version>           set in-repo requires to <version> and tidy (leaves tree dirty)
  tag  <version> [--push]  tag the current (clean) commit <subdir>/<version>, optionally push

Release flow: bump → review & commit → tag --push. See the header of this file.
EOF
    exit 2
    ;;
esac
