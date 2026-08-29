"""Unified env-lifecycle entrypoints.

    uv run test-setup --mode kubernetes|standalone
    uv run test-teardown --mode kubernetes|standalone

Provisioning lives here (not in pytest): bring the env up once, then run pytest
against it many times.
"""

from __future__ import annotations

import argparse

from setup import kubernetes, standalone


def setup_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-setup", description="Bring up a test environment")
    parser.add_argument("--mode", choices=("kubernetes", "standalone"), default="kubernetes")
    args = parser.parse_args(argv)
    return kubernetes.setup() if args.mode == "kubernetes" else standalone.setup()


def teardown_main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="test-teardown", description="Tear down a test environment")
    parser.add_argument("--mode", choices=("kubernetes", "standalone"), default="kubernetes")
    parser.add_argument(
        "--delete",
        action="store_true",
        help="destroy the cluster (cluster-delete) instead of stopping it (cluster-down)",
    )
    parser.add_argument(
        "--clean",
        action="store_true",
        help="standalone: also remove persistent data volumes",
    )
    args = parser.parse_args(argv)
    if args.mode == "kubernetes":
        return kubernetes.teardown(delete=args.delete)
    return standalone.teardown(clean=args.clean)
