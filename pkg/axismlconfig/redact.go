package axismlconfig

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Redacted returns a one-line-per-key dump of the resolved configuration with
// every secret field masked. Use it for startup logging — never log the raw
// Config, which holds plaintext secrets.
func Redacted(into any) string {
	fields := Walk(into)
	lines := make([]string, 0, len(fields))
	for _, f := range fields {
		val := "****"
		switch {
		case !f.Secret:
			val = fmt.Sprintf("%v", f.value.Interface())
		case f.value.Kind() == reflect.String && f.value.Len() == 0:
			val = "(unset)"
		}
		lines = append(lines, fmt.Sprintf("%s=%s", f.Path, val))
	}
	sort.Strings(lines)
	return strings.Join(lines, " ")
}
