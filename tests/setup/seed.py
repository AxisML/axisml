"""Make the Platform ``admin`` usable for the API/UI tests, deterministically.

The suite logs in as the default ``admin`` with a known suite password. Rather
than depend on the cluster's current admin password (which a prior run may have
rotated, and which can't be re-derived), env-setup resets the admin row directly
in the platform metadata DB: a fresh bcrypt hash of the suite password and
``must_change_password = false``. Go's bcrypt verifies Python's ``$2b$`` hashes.

This lives in env-setup (not a pytest fixture) on purpose — it is explicit,
runs once against a throwaway test cluster, and is fully idempotent.
"""

from __future__ import annotations

import time

import bcrypt
import httpx

from lib import config
from setup._proc import REPO_ROOT, log, run

LITE_COMPOSE = "axisml-lite/deploy/docker-compose.yaml"


def _admin_reset_sql(cfg: config.Config) -> str:
    hashed = bcrypt.hashpw(cfg.admin_password.encode(), bcrypt.gensalt()).decode()
    return (
        "UPDATE users SET password_hash = '{h}', must_change_password = false "
        "WHERE username = '{u}';".format(h=hashed, u=cfg.admin_username)
    )


def ensure_admin_ready() -> None:
    cfg = config.load()
    # No shell on the remote side: kubectl exec passes argv straight to the
    # container, so the bcrypt hash needs no escaping. The UPDATE is idempotent.
    run(
        [
            "kubectl", "exec", "-n", cfg.infra_namespace, cfg.db_pod, "--",
            "env", f"PGPASSWORD={cfg.db_password}",
            "psql", "-U", cfg.db_user, "-d", cfg.db_name, "-v", "ON_ERROR_STOP=1", "-c", _admin_reset_sql(cfg),
        ],
        cwd=REPO_ROOT,
    )
    log("seed", f"admin '{cfg.admin_username}' password reset; forced-change cleared")


def ensure_admin_ready_lite() -> None:
    """Lite variant: reset the admin row via ``docker compose exec`` into the
    compose Postgres. axisml-platform bootstraps the admin on first start, so we
    wait for its ``/readyz`` before rewriting the password to the suite's known
    credential (and clearing any forced change)."""
    cfg = config.load()
    _wait_platform_ready(cfg)
    run(
        [
            "docker", "compose", "-f", LITE_COMPOSE, "exec", "-T",
            "-e", f"PGPASSWORD={cfg.db_password}", "axisml-database",
            "psql", "-U", cfg.db_user, "-d", cfg.db_name, "-v", "ON_ERROR_STOP=1", "-c", _admin_reset_sql(cfg),
        ],
        cwd=REPO_ROOT,
    )
    log("seed", f"admin '{cfg.admin_username}' password reset; forced-change cleared")


def _wait_platform_ready(cfg: config.Config, *, timeout: float = 120.0) -> None:
    """Poll axisml-platform /readyz until it serves (migrations + bootstrap done)."""
    deadline = time.monotonic() + timeout
    last = "no response"
    while time.monotonic() < deadline:
        try:
            r = httpx.get(f"{cfg.lite_platform_url}/readyz", timeout=5.0)
            if r.status_code == 200:
                return
            last = f"HTTP {r.status_code}"
        except httpx.HTTPError as e:  # not up yet
            last = str(e)
        time.sleep(2.0)
    raise SystemExit(f"axisml-platform not ready at {cfg.lite_platform_url}/readyz ({last})")
