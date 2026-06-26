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
// axisml-system/docs/apis/*.yaml having been regenerated first. foldSystemSpecs
// below is the union that merges those surfaces together.
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
	"reflect"
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

// axisml-core mounts the three System modules' routers (Cluster Manager,
// Compute Service, Artifact Hub) on one HTTP server at the same paths the
// standalone System services expose. The composite OpenAPI document is
// therefore the UNION of those three surfaces plus the Lite-owned surface
// (probes + aggregate capability endpoint) built by buildDocument above.
//
// Each System surface is built in-process by its pkg/apidoc.Document builder —
// the SAME builder the System layer's own openapi-gen uses — so the fold is a
// direct Go-to-Go union with no YAML round-trip, and the System layer stays the
// single owner of those contracts (design §5).
//
// The fold is a plain union: the only cross-spec schema-name collisions are
// disambiguated at the source (each service's error schema is named per
// service — e.g. ComputeServiceError), so no renaming is needed here. Shared
// schemas with an identical definition (e.g. Corev1Toleration) deduplicate; a
// divergent same-name collision is a hard error pointing back at the source.

// litePaths are served by axisml-core itself, not delegated to a System module:
// the composed probes and the aggregate capability document. The System specs
// carry their own copies at these paths — we skip those and keep the Lite ones.
var litePaths = map[string]bool{
	"/healthz":             true,
	"/readyz":              true,
	"/api/v1/capabilities": true,
}

// liteOwnedSchemas are schema names the Lite document defines itself; the
// per-module System copies (e.g. each module's own Capabilities document) are
// dropped in favour of the Lite-owned definition.
var liteOwnedSchemas = map[string]bool{
	"Capabilities": true,
}

// foldSystemSpecs merges every System module's per-resource surface into dst.
func foldSystemSpecs(dst *openapigen.Document, srcs ...*openapigen.Document) error {
	for _, src := range srcs {
		if err := foldOne(dst, src); err != nil {
			return fmt.Errorf("fold %q: %w", src.Info.Title, err)
		}
	}
	return nil
}

func foldOne(dst, src *openapigen.Document) error {
	if dst.Components.Schemas == nil {
		dst.Components.Schemas = map[string]*openapigen.Schema{}
	}

	for name, schema := range src.Components.Schemas {
		if liteOwnedSchemas[name] {
			continue // Lite owns this name; drop the System copy.
		}
		if existing, present := dst.Components.Schemas[name]; present {
			if !reflect.DeepEqual(existing, schema) {
				return fmt.Errorf("schema %q has a divergent definition across specs; "+
					"give it a per-service name at the source", name)
			}
			continue // identical shared schema — deduplicate.
		}
		dst.Components.Schemas[name] = schema
	}

	for p, item := range src.Paths {
		if litePaths[p] {
			continue
		}
		if _, dup := dst.Paths[p]; dup {
			return fmt.Errorf("path %q already present in composite document", p)
		}
		dst.Paths[p] = item
	}

	mergeTags(dst, src)
	return nil
}

// mergeTags appends src's tag definitions that dst does not already declare,
// preserving src's order. Tags whose only operations were skipped (Health,
// Capabilities) are harmless extras and folded in for completeness.
func mergeTags(dst, src *openapigen.Document) {
	have := map[string]bool{}
	for _, t := range dst.Tags {
		have[t.Name] = true
	}
	for _, t := range src.Tags {
		if !have[t.Name] {
			dst.Tags = append(dst.Tags, t)
			have[t.Name] = true
		}
	}
}
