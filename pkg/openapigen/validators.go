package openapigen

import (
	"reflect"
	"strconv"
	"strings"
)

// PatternRule describes a custom validator/v10 tag that resolves to a regex
// pattern + length bounds on string fields. Per-service generators register
// these for their `binding:"axisml_name"` / `axisml_version` style rules.
type PatternRule struct {
	Tag       string // validator tag name, e.g. "axisml_name"
	Pattern   string // OpenAPI `pattern` regex
	MinLength int    // 0 = unset
	MaxLength int    // 0 = unset
}

// applyValidators reads the validator/v10 `binding` tag and translates the
// subset we use into OpenAPI Schema constraints. Unknown rules are skipped so
// the generator stays forward-compatible with new bindings. Pattern rules
// passed in via Options are matched first.
func applyValidators(s *Schema, binding string, t reflect.Type, patterns []PatternRule) {
	if binding == "" || s == nil {
		return
	}
	for _, raw := range strings.Split(binding, ",") {
		rule := strings.TrimSpace(raw)
		if rule == "" || rule == "required" {
			continue
		}
		if applyPatternRule(s, rule, patterns) {
			continue
		}
		switch {
		case strings.HasPrefix(rule, "min="):
			applySize(s, t, rule[len("min="):], minBound)
		case strings.HasPrefix(rule, "max="):
			applySize(s, t, rule[len("max="):], maxBound)
		case strings.HasPrefix(rule, "len="):
			applySize(s, t, rule[len("len="):], exactBound)
		case strings.HasPrefix(rule, "gte="):
			if v, ok := parseFloat(rule[len("gte="):]); ok {
				s.Minimum = &v
			}
		case strings.HasPrefix(rule, "lte="):
			if v, ok := parseFloat(rule[len("lte="):]); ok {
				s.Maximum = &v
			}
		}
	}
}

func applyPatternRule(s *Schema, rule string, patterns []PatternRule) bool {
	for _, p := range patterns {
		if p.Tag != rule {
			continue
		}
		if s == nil || s.Type != "string" {
			return true // matched a tag but not applicable to this type — don't fall through
		}
		if p.Pattern != "" {
			s.Pattern = p.Pattern
		}
		if p.MinLength > 0 {
			n := p.MinLength
			s.MinLength = &n
		}
		if p.MaxLength > 0 {
			n := p.MaxLength
			s.MaxLength = &n
		}
		return true
	}
	return false
}

type bound int

const (
	minBound bound = iota
	maxBound
	exactBound
)

// applySize maps validator/v10 size constraints (min/max/len) to the right
// OpenAPI keyword for the field's Go type: minLength on strings, minItems on
// slices/arrays/maps, minimum on numbers.
func applySize(s *Schema, t reflect.Type, raw string, b bound) {
	n, ok := parseInt(raw)
	if !ok {
		return
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		applyLengthBound(s, n, b)
	case reflect.Slice, reflect.Array, reflect.Map:
		applyItemsBound(s, n, b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		f := float64(n)
		applyNumberBound(s, f, b)
	}
}

func applyLengthBound(s *Schema, n int, b bound) {
	switch b {
	case minBound:
		s.MinLength = &n
	case maxBound:
		s.MaxLength = &n
	case exactBound:
		s.MinLength = &n
		s.MaxLength = &n
	}
}

func applyItemsBound(s *Schema, n int, b bound) {
	switch b {
	case minBound:
		s.MinItems = &n
	case maxBound:
		s.MaxItems = &n
	case exactBound:
		s.MinItems = &n
		s.MaxItems = &n
	}
}

func applyNumberBound(s *Schema, f float64, b bound) {
	switch b {
	case minBound:
		s.Minimum = &f
	case maxBound:
		s.Maximum = &f
	case exactBound:
		s.Minimum = &f
		s.Maximum = &f
	}
}

func parseInt(s string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
