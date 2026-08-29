"""Kubernetes deployment lifecycle for the black-box test suite."""

from __future__ import annotations

from setup import seed
from setup._proc import make, require, run, stage_runner

WORKLOAD_IMAGES = ["busybox:latest", "nginx:1.27"]
MINIKUBE_PROFILE = "axisml"


def _load_workload_images() -> None:
    for image in WORKLOAD_IMAGES:
        run(["docker", "pull", image])
        run(["minikube", "image", "load", image, "-p", MINIKUBE_PROFILE])


def setup() -> int:
    for tool in ("make", "docker", "minikube", "kubectl", "helm"):
        require(tool)
    return stage_runner(
        "test-setup [kubernetes]",
        [
            ("start minikube cluster", lambda: make("cluster-up")),
            ("build + load layer images", lambda: make("image-load")),
            ("load workload images", _load_workload_images),
            ("install helm layers", lambda: make("helm-install")),
            ("ensure admin usable", seed.ensure_admin_ready),
        ],
    )


def teardown(*, delete: bool = False) -> int:
    require("make")
    cluster_target = "cluster-delete" if delete else "cluster-down"
    return stage_runner(
        "test-teardown [kubernetes]",
        [
            ("uninstall helm layers", lambda: make("helm-uninstall")),
            (f"{cluster_target} minikube cluster", lambda: make(cluster_target)),
        ],
    )
