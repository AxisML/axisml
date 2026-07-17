// Package apispec exposes the generated Compute Service OpenAPI contract
// without pulling the reflection generator into the runtime module graph.
package apispec

import _ "embed"

//go:embed openapi.gen.yaml
var document []byte

// Document returns an isolated copy of the generated OpenAPI document.
func Document() []byte { return append([]byte(nil), document...) }
