# axisml-system/test/setup-envtest

This directory holds the shared `setup-envtest` binary used by every operator's
L1 integration target (the embedded apiserver+etcd is provided by
controller-runtime's envtest framework).

The binary itself is git-ignored. To install it:

```sh
make setup-envtest          # from the repo root
```

That target installs `sigs.k8s.io/controller-runtime/tools/setup-envtest` here
(version pinned in the top-level `Makefile`). The kubebuilder assets it
manages (`etcd`, `kube-apiserver`, `kubectl`) are stored in the
`setup-envtest`-default location (`~/.local/share/kubebuilder-envtest/` on
Linux; the equivalent under `~/Library/Application Support` on macOS), not
inside this directory.

Each operator's `Makefile` references `$(REPO_ROOT)/axisml-system/test/setup-envtest/setup-envtest`
when running `make <op>-integration`, so all three operators share one binary.
