// Command config-doc-gen renders the standalone configuration reference.
package main

import (
	"fmt"
	"strings"

	standalone "github.com/axisml/axisml/axisml-standalone"
	"github.com/axisml/axisml/axisml-standalone/internal/configutil"
)

func main() {
	fmt.Print(section("axisml-standalone", &standalone.Config{}))
}

func section(name string, into any) string {
	var b strings.Builder
	fprintf := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }
	fprintf("## %s\n\n", name)
	b.WriteString("Configuration source: **environment only**. This binary reads no config file.\n\n")
	b.WriteString("| Key | Environment variable | Default | Secret | Description |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, field := range configutil.Walk(into, standalone.DefaultEnvPrefix) {
		env, def, secret := "`"+field.EnvVar+"`", mdCode(field.Default), "—"
		if field.Secret {
			env, def, secret = "`"+field.EnvVar+"`<br>`"+field.EnvVar+"_FILE`", "—", "yes"
		}
		fprintf("| `%s` | %s | %s | %s | %s |\n", field.Path, env, def, secret, strings.ReplaceAll(field.Doc, "|", "\\|"))
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
