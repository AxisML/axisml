"""Run-unique, DNS-safe names so parallel/interrupted runs don't collide."""

from __future__ import annotations

import itertools
import os
import time

_seq = itertools.count(1)
# A short per-process suffix keeps names unique across concurrent invocations
# without needing Math.random()-style entropy.
_pid = os.getpid() % 1000


def unique_name(prefix: str) -> str:
    """e.g. ``e2e-run`` -> ``e2e-run-481-3-7``. Always starts with the prefix so
    leftover resources are reapable by an ``e2e-*`` / ``<prefix>-*`` selector."""
    return f"{prefix}-{int(time.time()) % 100000}-{_pid}-{next(_seq)}"
