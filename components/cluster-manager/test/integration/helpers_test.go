//go:build integration

package integration_test

import "strings"

// stringReader is io.NopCloser-equivalent for httptest body without
// pulling in extra deps.
func stringReader(s string) *strings.Reader { return strings.NewReader(s) }
