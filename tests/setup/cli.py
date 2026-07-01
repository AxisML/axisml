"""Unified env-lifecycle entrypoints.

    uv run test-setup     [--mode standard|lite]
    uv run test-teardown  [--mode standard|lite] [--delete] [--clean]

``--mode`` defaults to ``standard``. Provisioning lives here (not in pytest):
bring the env up once, then ``uv run pytest`` against it many times.
"""

from __future__ import annotations

import argparse

from setup import lite, standard


def setup_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-setup", description="Bring up a test environment")
    parser.add_argument("--mode", choices=["standard", "lite"], default="standard")
    args = parser.parse_args(argv)
    return standard.setup() if args.mode == "standard" else lite.setup()


def teardown_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-teardown", description="Tear down a test environment")
    parser.add_argument("--mode", choices=["standard", "lite"], default="standard")
    parser.add_argument(
        "--delete",
        action="store_true",
        help="standard: destroy the cluster (cluster-delete) instead of stopping it (cluster-down)",
    )
    parser.add_argument("--clean", action="store_true", help="lite: also remove the data volumes")
    args = parser.parse_args(argv)
    if args.mode == "standard":
        return standard.teardown(delete=args.delete)
    return lite.teardown(clean=args.clean)
