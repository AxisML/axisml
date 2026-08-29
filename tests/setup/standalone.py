"""Single-host Docker Compose lifecycle for the black-box test suite."""

from __future__ import annotations

from setup import seed
from setup._proc import make, require, stage_runner


def setup() -> int:
    for tool in ("make", "docker"):
        require(tool)
    return stage_runner(
        "test-setup [standalone]",
        [
            ("build images + start Compose stack", lambda: make("standalone-up")),
            ("ensure admin usable", seed.ensure_admin_ready_standalone),
        ],
    )


def teardown(*, clean: bool = False) -> int:
    require("make")
    extra = ["CLEAN=1"] if clean else []
    return stage_runner(
        "test-teardown [standalone]",
        [("stop Compose stack", lambda: make("standalone-down", *extra))],
    )
