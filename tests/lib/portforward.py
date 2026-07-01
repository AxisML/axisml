"""``kubectl port-forward`` subprocess management.

The System/Platform services are ClusterIP-only, so the Standard suite (and the
env-setup seeding) reach them over a local port-forward. Used as a context
manager so the forward is always torn down, and waits until the local port
actually accepts a TCP connection before returning (kubectl prints "Forwarding
from ..." before the tunnel is truly ready).
"""

from __future__ import annotations

import socket
import subprocess
import time
from contextlib import closing


def _free_port() -> int:
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def _accepts(host: str, port: int, timeout: float = 0.5) -> bool:
    try:
        with closing(socket.create_connection((host, port), timeout=timeout)):
            return True
    except OSError:
        return False


class PortForward:
    """A live ``kubectl port-forward`` to ``svc/<service>`` on a local port."""

    def __init__(self, service: str, namespace: str, remote_port: int, *, context: str | None = None):
        self.service = service
        self.namespace = namespace
        self.remote_port = remote_port
        self.context = context
        self.local_port = _free_port()
        self._proc: subprocess.Popen | None = None

    @property
    def local_url(self) -> str:
        return f"http://127.0.0.1:{self.local_port}"

    def start(self, ready_timeout: float = 30.0) -> "PortForward":
        cmd = ["kubectl", "port-forward", "-n", self.namespace, f"svc/{self.service}", f"{self.local_port}:{self.remote_port}"]
        if self.context:
            cmd += ["--context", self.context]
        self._proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        deadline = time.monotonic() + ready_timeout
        while time.monotonic() < deadline:
            if self._proc.poll() is not None:
                err = self._proc.stderr.read().decode() if self._proc.stderr else ""
                raise RuntimeError(f"port-forward to {self.service} exited early: {err}")
            if _accepts("127.0.0.1", self.local_port):
                return self
            time.sleep(0.25)
        self.stop()
        raise TimeoutError(f"port-forward to {self.service} not ready within {ready_timeout}s")

    def stop(self) -> None:
        if self._proc and self._proc.poll() is None:
            self._proc.terminate()
            try:
                self._proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self._proc.kill()
        self._proc = None

    def __enter__(self) -> "PortForward":
        return self.start()

    def __exit__(self, *exc) -> None:
        self.stop()
