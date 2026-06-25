// openapi-gen renders the OpenAPI 3.0.3 description of the artifacts HTTP API to
// axisml-system/docs/apis/artifact-hub.yaml. The document itself is built by
// pkg/apidoc (the single source of truth, shared with axisml-core); this command
// only renders it to YAML.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../docs/apis/artifact-hub.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axisml/axisml/components/artifact-hub/pkg/apidoc"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

func main() {
	out := flag.String("o", "../docs/apis/artifact-hub.yaml", "output path")
	v := flag.String("version", defaultVersion, "info.version field")
	flag.Parse()

	doc := apidoc.Document(*v)
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
