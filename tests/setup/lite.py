"""Lite-form (single-host Docker Compose) environment lifecycle.

Builds the images and brings up / tears down the Lite compose stack: PostgreSQL +
axisml-core (System modules, :8090) + axisml-platform (API + SPA, :8080). Driven
from ``setup.cli`` via ``test-setup --mode lite`` / ``test-teardown --mode lite``.

Lite has no Kubernetes and is fixed to a single static tenant (``default``), but
it ships the Platform layer and UI, so the admin is seeded (password reset to the
suite's known credential) just like the Standard form.
"""

from __future__ import annotations

from setup import seed
from setup._proc import make, require, stage_runner


def setup() -> int:
    for tool in ("make", "docker"):
        require(tool)
    return stage_runner(
        "test-setup [lite]",
        # `make lite-up` builds the axisml-core + axisml-platform images as part of compose up.
        [
            ("build images + start compose stack (core :8090, platform :8080)", lambda: make("lite-up")),
            ("ensure admin usable (clear forced password change)", seed.ensure_admin_ready_lite),
        ],
    )


def teardown(*, clean: bool = False) -> int:
    require("make")
    extra = ["CLEAN=1"] if clean else []
    return stage_runner(
        "test-teardown [lite]",
        [("stop + remove compose stack", lambda: make("lite-down", *extra))],
    )
