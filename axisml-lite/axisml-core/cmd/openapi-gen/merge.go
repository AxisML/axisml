package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/axisml/axisml/pkg/openapigen"
	"sigs.k8s.io/yaml"
)

// axisml-core mounts the three System modules' routers (Cluster Manager,
// Compute Service, Artifact Hub) on one HTTP server at the same paths the
// standalone System services expose. The composite OpenAPI document is
// therefore the UNION of those three generated specs plus the Lite-owned
// surface (probes + aggregate capability endpoint) already built in main.go.
//
// We fold by re-reading the System specs' generated YAML (each produced by the
// SAME pkg/openapigen Document type, so the round-trip is lossless) rather than
// re-reflecting their Go DTOs — the System layer stays the single owner of
// those contracts (design §5).
//
// The fold is a plain union: the only cross-spec schema-name collisions are
// disambiguated at the source (each service's error schema is named per
// service — e.g. ComputeServiceProblem), so no renaming is needed here. Shared
// schemas with an identical definition (e.g. Corev1Toleration) deduplicate; a
// divergent same-name collision is a hard error pointing back at the source.

var systemSpecFiles = []string{"cluster-manager", "compute-service", "artifact-hub"}

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
func foldSystemSpecs(dst *openapigen.Document, dir string) error {
	for _, name := range systemSpecFiles {
		path := filepath.Join(dir, name+".yaml")
		src, err := loadSpec(path)
		if err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
		if err := foldOne(dst, src); err != nil {
			return fmt.Errorf("fold %s: %w", path, err)
		}
	}
	return nil
}

func loadSpec(path string) (*openapigen.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc openapigen.Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
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
