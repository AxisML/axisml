// Command config-doc-gen prints the platform-backend configuration reference
// section (Markdown) for assembly into docs/configuration.md. The content is
// derived from the Config struct's tags, so it stays in sync with the loader.
package main

import (
	"fmt"

	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
	"github.com/axisml/axisml/pkg/configdoc"
)

func main() {
	fmt.Print(configdoc.Section("platform-backend", &config.Config{}, false))
}
