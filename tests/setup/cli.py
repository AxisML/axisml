"""Unified env-lifecycle entrypoints.

    uv run test-setup
    uv run test-teardown [--delete]

Provisioning lives here (not in pytest): bring the env up once, then run pytest
against it many times.
"""

from __future__ import annotations

import argparse

from setup import standard


def setup_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-setup", description="Bring up a test environment")
    parser.parse_args(argv)
    return standard.setup()


def teardown_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-teardown", description="Tear down a test environment")
    parser.add_argument(
        "--delete",
        action="store_true",
        help="destroy the cluster (cluster-delete) instead of stopping it (cluster-down)",
    )
    args = parser.parse_args(argv)
    return standard.teardown(delete=args.delete)
