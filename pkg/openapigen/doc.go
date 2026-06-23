// Package openapigen renders OpenAPI 3.0.3 specs from Go request/response
// structs by reflection. It is consumed by per-service cmd/openapi-gen tools
// in axisml-system/compute-service, axisml-system/artifact-hub, and axisml-system/cluster-manager.
//
// The engine is route-agnostic: each service is responsible for hand-building
// its `paths:` table (single source of truth, reviewable in PRs) and
// registering its component schemas. The reflection engine handles type
// expansion, $ref management, validator-tag translation, and well-known type
// substitutions.
package openapigen
