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

import bcrypt

from lib import config
from setup._proc import REPO_ROOT, log, run

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
