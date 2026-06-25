// openapi-gen renders the COMPLETE OpenAPI 3.0.3 description of the axisml-core
// HTTP surface to axisml-lite/docs/apis/axisml-core.yaml.
//
// axisml-core composes the three System modules (Cluster Manager + Compute
// Service + Artifact Hub) onto one HTTP server at the same paths the standalone
// System services expose, plus its own probes and aggregate capability
// endpoint. The composite spec is the UNION of the Lite-owned surface (built
// here by reflection over internal/core) and the three System surfaces.
//
// We build each System surface in-process from its pkg/apidoc.Document builder —
// the SAME builder the System layer's own openapi-gen renders to YAML — so the
// System layer stays the single owner of those contracts (design §5) while the
// composite is assembled directly from Go: no YAML re-read, and no dependency on
// axisml-system/docs/apis/*.yaml having been regenerated first. See merge.go for
// the fold.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../docs/apis/axisml-core.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axisml/axisml/axisml-lite/axisml-core/internal/core"
	arthubapidoc "github.com/axisml/axisml/components/artifact-hub/pkg/apidoc"
	clustermgrapidoc "github.com/axisml/axisml/components/cluster-manager/pkg/apidoc"
	computeapidoc "github.com/axisml/axisml/components/compute-service/pkg/apidoc"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

const (
	tagCapabilities = "Capabilities"
	tagHealth       = "Health"
)

func main() {
	out := flag.String("o", "../../docs/apis/axisml-core.yaml", "output path")
	v := flag.String("version", defaultVersion, "info.version field")
	flag.Parse()

	doc := buildDocument(*v)
	// Fold the three System surfaces in-process from the SAME pkg/apidoc builders
	// the System layer's own openapi-gen uses — no YAML re-read, no dependency on
	// the System specs having been regenerated first.
	if err := foldSystemSpecs(doc,
		clustermgrapidoc.Document(*v),
		computeapidoc.Document(*v),
		arthubapidoc.Document(*v),
	); err != nil {
		fail("fold system specs: %v", err)
	}
	data, err := openapigen.MarshalYAML(doc)
	if err != nil {
		fail("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail("write: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "openapi-gen: "+format+"\n", args...)
	os.Exit(1)
}

func buildDocument(version string) *openapigen.Document {
	g := openapigen.New(openapigen.Options{
		// The capability DTO lives in the flat internal/core package, so map it to
		// an empty prefix and the schema name equals the Go type name verbatim.
		PackageNamer: func(pkg string) (string, bool) {
			if strings.HasSuffix(pkg, "/axisml-lite/axisml-core/internal/core") {
				return "", true
			}
			return "", false
		},
	})

	g.Register("Capabilities", core.Capabilities{}, openapigen.ResponseMode)

	tags := []openapigen.TagEntry{
		{Name: tagCapabilities, Description: "Aggregate deployment-form capability document — the three System modules' per-form documents folded under their component key (design §5.5)."},
		{Name: tagHealth, Description: "Liveness and readiness probes."},
	}

	paths := map[string]openapigen.PathItem{}

	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": openapigen.StringResp("ok")},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{
			"200": openapigen.StringResp("ok"),
			"503": openapigen.StringResp("dependency not yet ready"),
		},
	}}
	paths["/api/v1/capabilities"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagCapabilities}, Summary: "Get the aggregate capability document", OperationID: "getCapabilities",
		Responses: map[string]openapigen.Response{"200": openapigen.JSONResp("Aggregate capability document.", "Capabilities")},
	}}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:   "AxisML Core API",
			Version: version,
			Description: "Complete HTTP surface of axisml-core (AxisML Lite single-host form): " +
				"Cluster Manager + Compute Service + Artifact Hub served on one HTTP server, " +
				"plus the Standalone Docker runtime. This document folds the three System specs " +
				"(axisml-system/docs/apis/{cluster-manager,compute-service,artifact-hub}.yaml) " +
				"into one — their per-resource contracts are reachable at the same paths here — " +
				"alongside the Lite-owned probes and aggregate capability endpoint.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin (axisml-core)"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
