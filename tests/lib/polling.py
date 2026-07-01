"""Polling helper for eventual conditions (mirrors the Go suite's `eventually`)."""

from __future__ import annotations

import time
from collections.abc import Callable
from typing import TypeVar

T = TypeVar("T")


class _NotReady(Exception):
    pass


def retry(msg: str = "condition not met") -> None:
    """Raise inside an `eventually` callback to signal 'not ready, keep polling'."""
    raise _NotReady(msg)


def eventually(fn: Callable[[], T], *, timeout: float, interval: float = 2.0) -> T:
    """Poll ``fn`` until it returns without raising, or ``timeout`` (s) elapses.

    The callback signals 'not yet' by raising — either via ``retry()`` or any
    AssertionError. Any other exception propagates immediately (a real failure).
    The last not-ready message is surfaced on timeout.
    """
    deadline = time.monotonic() + timeout
    last = "no attempt made"
    while time.monotonic() < deadline:
        try:
            return fn()
        except (_NotReady, AssertionError) as e:
            last = str(e) or e.__class__.__name__
            time.sleep(interval)
    raise TimeoutError(f"eventually: timed out after {timeout}s: {last}")
