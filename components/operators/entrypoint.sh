#!/bin/sh
set -eu

operator="${AXISML_OPERATOR:-}"

case "${1:-}" in
  tenant-operator|mljob-operator|mlservice-operator)
    operator="$1"
    shift
    ;;
esac

case "$operator" in
  tenant-operator|mljob-operator|mlservice-operator)
    exec "/usr/local/bin/$operator" "$@"
    ;;
  "")
    echo "missing operator name; pass tenant-operator, mljob-operator, or mlservice-operator as the first arg, or set AXISML_OPERATOR" >&2
    exit 64
    ;;
  *)
    echo "unknown operator: $operator" >&2
    exit 64
    ;;
esac
