package main

import "github.com/axisml/axisml/pkg/openapigen"

// obj is a terse alias for the JSON-object shape of a whole-object example.
// Examples are authored as plain Go values (maps / slices / scalars) that
// marshal to the same JSON the API emits; the generator attaches each one to
// its schema's `example`, and the frontend mock codegen reads them back to
// build fixtures (see axisml-platform/frontend/src/api/mock).
type obj = map[string]any

// Timestamps used across examples so every fixture shares one coherent clock.
const (
	exCreatedAt  = "2026-06-20T08:00:00Z"
	exUpdatedAt  = "2026-06-28T09:30:00Z"
	exStartedAt  = "2026-06-28T09:00:00Z"
	exFinishedAt = "2026-06-28T09:25:00Z"
)

// registerExamples attaches a whole-object example to every component schema.
// One function per server-file domain keeps the examples co-located with the
// DTOs they mirror and lets the files be edited independently. Each helper
// calls g.SetExample, which panics on an unknown schema name so a rename can't
// silently drop an example.
func registerExamples(g *openapigen.Generator) {
	exCommon(g)
	exAuth(g)
	exTenant(g)
	exWorkspace(g)
	exJob(g)
	exMLService(g)
	exTraffic(g)
	exResourcePool(g)
	exDataVolume(g)
	exArtifact(g)
	exExperiment(g)
	exProxy(g)
}
