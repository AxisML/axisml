// Command config-doc-gen prints the artifact-hub configuration reference
// section (Markdown) for assembly into docs/configuration.md. The content is
// derived from the Config struct's tags, so it stays in sync with the loader.
package main

import (
	"fmt"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/config"
	"github.com/axisml/axisml/pkg/configdoc"
)

func main() {
	fmt.Print(configdoc.Section("artifact-hub", &config.Config{}, false))
}
