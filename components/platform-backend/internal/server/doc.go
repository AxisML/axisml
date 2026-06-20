// Package server declares the AxisML Platform HTTP API surface as Go DTO types.
//
// Platform is not yet implemented; this package is a contract-only "shell".
// Its sole consumer is cmd/openapi-gen, which reflects these structs into
// docs/openapi/platform.yaml via pkg/openapigen — the same code-first engine
// the cluster-manager / compute-service / artifact-hub specs use. Keeping the
// surface in Go (rather than a hand-maintained YAML) means the spec is
// generated and guarded by `make doc-test`.
//
// Conventions (mirrored by cmd/openapi-gen's generator options):
//
//   - Every exported struct's name is its OpenAPI component name. All DTOs live
//     in this one package so the generator's PackageNamer can map the package
//     path to an empty prefix, making component name == Go type name.
//   - Scalar string formats (uuid / email / password / uri) are modelled as the
//     named string types below and rendered inline via WellKnown.
//   - Enums are named string types; see enums.go. The generator decides whether
//     each renders as a referenced component or an inline enum.
//   - Required vs optional: response fields are plain when always present and
//     carry `,omitempty` when optional; genuinely nullable fields use a pointer.
//     Request fields use `binding:"required"` for required entries.
package server

// StringMap is a flat string→string map (labels, annotations, nodeSelector).
type StringMap map[string]string

// ResourceMap is a Kubernetes-style resource quantity map, e.g.
// {"cpu": "100", "memory": "1Ti", "nvidia.com/gpu": "8"}.
type ResourceMap map[string]string

// Toleration mirrors a Kubernetes corev1.Toleration (free-form pass-through).
type Toleration map[string]any

// UUID renders as `string` with `format: uuid`.
type UUID string

// Email renders as `string` with `format: email`.
type Email string

// Password renders as `string` with `format: password`.
type Password string

// URI renders as `string` with `format: uri`.
type URI string
