# Embedding axisml-core

This directory is a standalone Go project that embeds `axisml-core` as a
library. It owns the HTTP server, mounts the AxisML API under `/axisml`, and
serves a host-owned route at `/host/ping`.

The project shape is intentionally copyable:

- `go.mod` depends on `github.com/axisml/axisml/axisml-lite/axisml-core`.
- `main.go` uses only the public `pkg/core` API.
- `config/` contains the Lite `ResourcePool` and `Tenant` YAML read at startup.

## Build

From this directory:

```sh
go mod tidy
go build -o bin/axisml-core-embed .
```

Inside the AxisML repository you can also run:

```sh
make -C axisml-lite/examples
```

## Run

The embedded core still needs the same runtime services as the standalone Lite
binary:

- Docker Engine, reachable through the local socket or `DOCKER_HOST`;
- PostgreSQL, reachable through the `AXISML_DATABASE_*` variables;
- an OCI registry endpoint for Artifact Hub operations.

For a local smoke test, start PostgreSQL and run this host app against it:

```sh
docker run --name axisml-core-embed-postgres --rm -d \
  -e POSTGRES_USER=axisml \
  -e POSTGRES_PASSWORD=axisml \
  -e POSTGRES_DB=axisml \
  -p 15432:5432 \
  postgres:16

export AXISML_DATABASE_HOST=localhost
export AXISML_DATABASE_PORT=15432
export AXISML_DATABASE_NAME=axisml
export AXISML_DATABASE_USER=axisml
export AXISML_DATABASE_PASSWORD=axisml
export AXISML_OCI_ENDPOINT=http://localhost:5000

go run .
```

Then call both the embedded AxisML API and the host API:

```sh
curl http://localhost:9090/axisml/api/v1/capabilities
curl http://localhost:9090/host/ping
```

## Updating axisml-core

When this example is kept outside the AxisML repository, point the `replace`
directives in `go.mod` at the AxisML checkout you want to embed. To update with
that model, pull or check out the newer AxisML revision, keep the `replace`
paths pointing at it, then run:

```sh
go mod tidy
go build -o bin/axisml-core-embed .
```

If `axisml-core` and its sibling modules are later published as normal module
versions, remove the local `replace` directives and update the requirement with
`go get github.com/axisml/axisml/axisml-lite/axisml-core@<version>`.
