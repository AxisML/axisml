// openapi-gen renders the OpenAPI 3.0.3 description of the artifacts HTTP API to
// axisml-system/docs/apis/artifact-hub.yaml. The document itself is built by
// pkg/apidoc (the single source of truth); this command
// only renders it to YAML.
//
// Run from the pkg/apidoc tool module:
//
//	go run ./cmd/openapi-gen -o ../../../docs/apis/artifact-hub.yaml -embed ../apispec/openapi.gen.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axisml/axisml/axisml-system/artifact-hub/pkg/apidoc"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

func main() {
	out := flag.String("o", "../../../docs/apis/artifact-hub.yaml", "output path")
	embed := flag.String("embed", "../apispec/openapi.gen.yaml", "runtime module embed output path")
	v := flag.String("version", defaultVersion, "info.version field")
	flag.Parse()

	doc := apidoc.Document(*v)
	data, err := openapigen.MarshalYAML(doc)
	if err != nil {
		fail("marshal: %v", err)
	}
	for _, path := range []string{*out, *embed} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fail("write %s: %v", path, err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "openapi-gen: "+format+"\n", args...)
	os.Exit(1)
}
