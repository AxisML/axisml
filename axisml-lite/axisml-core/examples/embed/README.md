# Embedding axisml-core

A runnable example of hosting axisml-core **inside another Go program** instead
of running the `axisml-core` binary standalone. It imports `pkg/core`, mounts the
full axisml-core HTTP API under a path prefix on the host's own server, and
drives the module background loops from a context the host owns.

## The embedding API

```go
app, err := core.New(ctx, cfg,
    core.WithSettings(settings),         // override bind addr / paths / network
    core.WithStaticConfig(staticConfig), // supply ResourcePool + Tenant in memory
    // core.WithDB(db),                  // reuse a *gorm.DB you already own
    // core.WithLogger(log),             // supply your own logr.Logger
)
defer app.Close()

app.Migrate()                                    // bring the schema up to date

mux.Handle("/axisml/", http.StripPrefix("/axisml", app.Handler())) // mount the API
for _, r := range app.Runnables() { go r.Start(ctx) }              // reconcilers + GC
```

`New` does pure assembly — no env reads, no `/etc/axisml/pools` requirement, no
`os.Exit`. The host owns the listener and lifecycle. (`app.Serve(ctx)` is the
all-in-one alternative the standalone binary uses: it starts the runnables and a
server on `Settings.APIBindAddress` and blocks.)

axisml-core's routes live under `/api/v1`, so behind the `/axisml` strip-prefix
they surface at `/axisml/api/v1/...`.

## Running it

The host process must provide the two runtime dependencies axisml-core needs:

- a reachable **Docker daemon** (`DOCKER_HOST`, or the default socket) — the
  Standalone compute backend,
- a **PostgreSQL** the `Config` points at.

```sh
export PGHOST=localhost PGUSER=axisml PGPASSWORD=axisml PGDATABASE=axisml
go run ./examples/embed
# host server listening on :9090 — axisml-core API under /axisml/api/v1

curl localhost:9090/axisml/api/v1/capabilities   # axisml-core (unauthenticated)
curl localhost:9090/host/ping                    # the host's own route
```

Without Docker + Postgres, `New` fails fast — but the example always compiles, so
it doubles as a compile-checked usage reference.

## Consuming axisml-core from a separate repository

This example lives inside the axisml module, so it imports `pkg/core` directly.
A **separate** repo additionally has to resolve axisml-core's dependency graph:
its `go.mod` replaces the five `components/*` modules, `pkg/openapigen`, and pins
`k8s.io/kube-openapi`. Pull axisml in as a git submodule (or vendor it) and
replicate that `replace` block in the host `go.mod`, pointing the relative paths
at the submodule.
