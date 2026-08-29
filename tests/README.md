# AxisML black-box test suite

The Python/pytest suite drives AxisML only through generated HTTP clients and
the Platform UI. The same tests run against both deployment forms; white-box
Kubernetes and Docker assertions stay in Go integration tests.

## Install

```sh
cd tests
uv sync
uv run playwright install chromium
```

## Environment lifecycle

Provisioning is explicit and separate from pytest:

```sh
uv run test-setup --mode kubernetes
uv run test-teardown --mode kubernetes
uv run test-teardown --mode kubernetes --delete

uv run test-setup --mode standalone
uv run test-teardown --mode standalone
uv run test-teardown --mode standalone --clean
```

Kubernetes builds and loads images into minikube, installs the three Helm layers
and uses port-forwards. Standalone builds the top-level `axisml-standalone`
distribution and Platform images from the current checkout, then starts
`axisml-standalone/compose.yaml`.

## Run

Pass the same mode used for environment setup:

```sh
uv run pytest --mode kubernetes api
uv run pytest --mode kubernetes e2e

uv run pytest --mode standalone api
uv run pytest --mode standalone e2e
```

Unmarked tests are shared. `kubernetes_only` and `standalone_only` express a
hard runtime boundary; finer differences use the capability matrix in
`lib/harness.py`.

## Generated clients

Clients are committed and generated from the canonical specs in this repository:

```sh
make -C .. client-gen
```

Regenerate after changing a component or Platform HTTP contract. The aggregate
standalone capability contract is tested directly because it intentionally
wraps the three component capability documents.
