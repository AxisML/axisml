"""Standard-form (Kubernetes) environment lifecycle.

Brings up / tears down the full Standard stack the API + UI suites need: a
minikube cluster, freshly built+loaded layer images, the three Helm layers,
workload images, and a usable admin account. Driven from ``setup.cli`` via
``test-setup --mode standard`` / ``test-teardown --mode standard``.

Pure provisioning over the existing Makefile tooling — no pytest, no test imports.
"""

from __future__ import annotations

from setup import seed
from setup._proc import make, require, run, stage_runner

# Workload images the compute tests run (busybox MLRun / nginx MLService). Loaded
# into minikube so pods schedule offline (imagePullPolicy: IfNotPresent).
WORKLOAD_IMAGES = ["busybox:latest", "nginx:1.27"]
MINIKUBE_PROFILE = "axisml"


def _load_workload_images() -> None:
    for img in WORKLOAD_IMAGES:
        run(["docker", "pull", img])
        run(["minikube", "image", "load", img, "-p", MINIKUBE_PROFILE])


def setup() -> int:
    for tool in ("make", "docker", "minikube", "kubectl", "helm"):
        require(tool)
    return stage_runner(
        "test-setup [standard]",
        [
            ("start minikube cluster", lambda: make("cluster-up")),
            ("build + load layer images", lambda: make("image-load")),
            ("load workload images", _load_workload_images),
            ("install helm layers (infra -> system -> platform)", lambda: make("helm-install")),
            ("ensure admin usable (clear forced password change)", seed.ensure_admin_ready),
        ],
    )


def teardown(*, delete: bool = False) -> int:
    require("make")
    cluster_target = "cluster-delete" if delete else "cluster-down"
    return stage_runner(
        "test-teardown [standard]",
        [
            ("uninstall helm layers (platform -> system -> infra)", lambda: make("helm-uninstall")),
            (f"{cluster_target} minikube cluster", lambda: make(cluster_target)),
        ],
    )
