package server

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// JSONLabelsSQL renders the K8s label selector into a Postgres `WHERE`
// fragment over a `labels jsonb` column. Mirrors compute-service's helper
// (covers =, ==, !=, exists, !exists, in (…), notin (…) per K8s grammar).
// Empty input → empty fragment (match-all).
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
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IS NOT NULL", column))
			args = append(args, r.Key())
		case "!":
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IS NULL", column))
			args = append(args, r.Key())
		case "in":
			vals := r.Values().UnsortedList()
			if len(vals) == 0 {
				continue
			}
			placeholders := make([]string, len(vals))
			for i := range vals {
				placeholders[i] = "?"
			}
			clauses = append(clauses, fmt.Sprintf("(%s ->> ?) IN (%s)", column, strings.Join(placeholders, ",")))
			args = append(args, r.Key())
			for _, v := range vals {
				args = append(args, v)
			}
		case "notin":
			vals := r.Values().UnsortedList()
			if len(vals) == 0 {
				continue
			}
			placeholders := make([]string, len(vals))
			for i := range vals {
				placeholders[i] = "?"
			}
			clauses = append(clauses, fmt.Sprintf("((%s ->> ?) IS NULL OR (%s ->> ?) NOT IN (%s))",
				column, column, strings.Join(placeholders, ",")))
			args = append(args, r.Key(), r.Key())
			for _, v := range vals {
				args = append(args, v)
			}
		}
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	return strings.Join(clauses, " AND "), args, nil
}
