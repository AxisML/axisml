// Command config-doc-gen prints the axisml-core (Lite) configuration reference
// section (Markdown) for assembly into docs/configuration.md. axisml-core reads
// no config file (env-only), so the section is rendered with envOnly=true.
package main

import (
	"fmt"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
	"github.com/axisml/axisml/pkg/configdoc"
)

func main() {
	fmt.Print(configdoc.Section("axisml-core (Lite)", &core.Config{}, true))
}
