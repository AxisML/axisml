"""Shared helpers for the environment-provisioning entrypoints.

These scripts orchestrate the existing Makefile/minikube/helm tooling via
subprocess — they are NOT pytest fixtures and never import the test code. Output
is streamed live so a `uv run test-setup` reads like running the make targets by
hand, and any failing stage aborts with a clear, non-zero exit.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

# tests/setup/_proc.py -> parents[2] == repo root (the dir holding the root Makefile).
REPO_ROOT = Path(__file__).resolve().parents[2]


def log(stage: str, msg: str) -> None:
    print(f"\n\033[1m[{stage}]\033[0m {msg}", flush=True)


def run(cmd: list[str], *, cwd: Path | None = None, env: dict[str, str] | None = None) -> None:
    """Run a command, streaming output; raise SystemExit on non-zero exit."""
    printable = " ".join(cmd)
    print(f"  $ {printable}", flush=True)
    result = subprocess.run(cmd, cwd=str(cwd or REPO_ROOT), env=env)
    if result.returncode != 0:
        raise SystemExit(f"command failed ({result.returncode}): {printable}")


def make(target: str, *args: str) -> None:
    """Invoke a root-Makefile target."""
    run(["make", target, *args])


def require(tool: str) -> None:
    """Fail fast with a friendly message if a required CLI is missing."""
    if shutil.which(tool) is None:
        raise SystemExit(f"required tool not found on PATH: {tool}")


def stage_runner(name: str, stages: list[tuple[str, callable]]) -> int:
    """Run ordered (label, fn) stages, printing progress; return a process exit code."""
    log(name, f"{len(stages)} stage(s)")
    for i, (label, fn) in enumerate(stages, 1):
        log(name, f"({i}/{len(stages)}) {label}")
        fn()
    log(name, "done ✓")
    return 0
