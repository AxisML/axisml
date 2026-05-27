package server

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// ParseLabelSelector turns the raw ?labelSelector= query into something the
// caller can use to filter rows whose `labels jsonb` column matches the
// K8s selector grammar (=, ==, !=, in, notin, key, !key — comma-separated
// AND). Empty input means "match everything".
//
// The returned predicate captures the parsed selector and tests against a
// flat map[string]string view of the row's labels.
func ParseLabelSelector(raw string) (func(map[string]string) bool, error) {
	if strings.TrimSpace(raw) == "" {
		return func(map[string]string) bool { return true }, nil
	}
	sel, err := labels.Parse(raw)
	if err != nil {
		return nil, apperrors.Newf(apperrors.CodeValidation, "labelSelector parse: %v", err)
	}
	return func(m map[string]string) bool {
		return sel.Matches(labels.Set(m))
	}, nil
}

// JSONLabelsSQL renders the K8s label selector into a Postgres `WHERE`
// fragment over a `labels jsonb` column. It only supports the subset
// that's safely expressible in SQL (equality, inequality, exists/!exists);
// `in`/`notin` selectors fall back to caller-side filtering via the
// predicate returned from ParseLabelSelector.
//
// The returned (sqlFragment, args) is empty for a match-all selector.
func JSONLabelsSQL(column, raw string) (string, []any, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil, nil
	}
	sel, err := labels.Parse(raw)
	if err != nil {
		return "", nil, apperrors.Newf(apperrors.CodeValidation, "labelSelector parse: %v", err)
	}
	reqs, _ := sel.Requirements()
	var clauses []string
	var args []any
	for _, r := range reqs {
		switch r.Operator() {
		case "=", "==":
			vals := r.Values().UnsortedList()
			if len(vals) != 1 {
				continue
			}
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) = ?", column))
			args = append(args, r.Key(), vals[0])
		case "!=":
			vals := r.Values().UnsortedList()
			if len(vals) != 1 {
				continue
			}
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IS DISTINCT FROM ?", column))
			args = append(args, r.Key(), vals[0])
		case "exists":
			// `jsonb ? text` clashes with gorm's `?` placeholder; use the
			// equivalent (column ->> key) IS NOT NULL form.
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IS NOT NULL", column))
			args = append(args, r.Key())
		case "!":
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IS NULL", column))
			args = append(args, r.Key())
		}
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	return strings.Join(clauses, " AND "), args, nil
}
