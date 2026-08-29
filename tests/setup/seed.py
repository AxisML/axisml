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

STANDALONE_COMPOSE = "axisml-standalone/compose.yaml"

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


def ensure_admin_ready_standalone() -> None:
    """Reset the Compose Platform admin to the suite credential."""
    cfg = config.load()
    _wait_platform_ready(cfg)
    run(
        [
            "docker", "compose", "-f", STANDALONE_COMPOSE, "exec", "-T",
            "-e", f"PGPASSWORD={cfg.db_password}", "axisml-database",
            "psql", "-U", cfg.db_user, "-d", cfg.db_name,
            "-v", "ON_ERROR_STOP=1", "-c", _admin_reset_sql(cfg),
        ],
        cwd=REPO_ROOT,
    )
    log("seed", f"admin '{cfg.admin_username}' password reset; forced-change cleared")


def _wait_platform_ready(cfg: config.Config, *, timeout: float = 120.0) -> None:
    deadline = time.monotonic() + timeout
    last = "no response"
    while time.monotonic() < deadline:
        try:
            response = httpx.get(f"{cfg.standalone_platform_url}/readyz", timeout=5.0)
            if response.status_code == 200:
                return
            last = f"HTTP {response.status_code}"
        except httpx.HTTPError as error:
            last = str(error)
        time.sleep(2.0)
    raise SystemExit(
        f"axisml-platform not ready at {cfg.standalone_platform_url}/readyz ({last})"
    )
