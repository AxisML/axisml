// Command config-doc-gen prints the artifact-hub configuration reference
// section (Markdown) for assembly into docs/configuration.md. The content is
// derived from the Config struct's tags, so it stays in sync with the loader.
package main

import (
	"fmt"
	"strings"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/config"
)

func main() {
	fmt.Print(section("artifact-hub", &config.Config{}))
}

func section(name string, into any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", name)
	b.WriteString("Config file: `/etc/axisml/config.yaml` (override with `--config` or `AXISML_CONFIG`). Every key is also overridable by its `AXISML_` variable.\n\n")
	b.WriteString("| Key | Environment variable | Default | Secret | Description |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, field := range config.Walk(into) {
		env, def, secret := "`"+field.EnvVar+"`", mdCode(field.Default), "—"
		if field.Secret {
			env, def, secret = "`"+field.EnvVar+"`<br>`"+field.EnvVar+"_FILE`", "—", "yes"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", field.Path, env, def, secret, strings.ReplaceAll(field.Doc, "|", "\\|"))
	}
	b.WriteString("\n")
	return b.String()
}

func mdCode(value string) string {
	if value == "" {
		return "—"
	}
	return "`" + value + "`"
}
