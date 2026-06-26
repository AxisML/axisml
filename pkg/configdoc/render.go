// Package configdoc renders a service's configuration reference section from its
// Config struct, using the same struct tags the loader reads (axismlconfig).
// Each service's cmd/config-doc-gen prints its section; a repo target assembles
// them with a static preamble into docs/configuration.md.
package configdoc

import (
	"fmt"
	"strings"

	"github.com/axisml/axisml/pkg/axismlconfig"
)

// Section renders the Markdown reference for one service. When envOnly is true
// the service reads no config file (Lite axisml-core) and the keys are supplied
// purely as AXISML_ environment variables.
func Section(name string, into any, envOnly bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", name)
	if envOnly {
		b.WriteString("Configuration source: **environment only** — this binary reads no config file. " +
			"Each key below is supplied as its `AXISML_` variable.\n\n")
	} else {
		b.WriteString("Config file: `/etc/axisml/config.yaml` (override with `--config` or `AXISML_CONFIG`). " +
			"Every key is also overridable by its `AXISML_` variable.\n\n")
	}
	b.WriteString("| Key | Environment variable | Default | Secret | Description |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, f := range axismlconfig.Walk(into) {
		env := "`" + f.EnvVar + "`"
		def := mdCode(f.Default)
		secret := "—"
		if f.Secret {
			env = "`" + f.EnvVar + "`<br>`" + f.EnvVar + "_FILE`"
			def = "—"
			secret = "yes"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", f.Path, env, def, secret, mdCell(f.Doc))
	}
	b.WriteString("\n")
	return b.String()
}

func mdCode(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + s + "`"
}

// mdCell escapes pipe characters so free text does not break the table.
func mdCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
