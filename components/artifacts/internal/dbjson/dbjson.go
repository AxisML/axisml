// Package dbjson centralises the small set of helpers that artifacts'
// repositories share for marshalling map columns and detecting PG error
// codes. Each helper has exactly two consumers today (repo + artifact);
// the package exists so a future Kind handler doesn't drift away from
// the established shape.
package dbjson

import (
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
)

// MapToJSON marshals a string map into a jsonb-bound column value. Empty
// maps render as `{}` so the database always sees a valid jsonb literal.
func MapToJSON(m map[string]string) (datatypes.JSON, error) {
	if len(m) == 0 {
		return datatypes.JSON([]byte("{}")), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

// DecodeStringMap parses a jsonb column into a string map, swallowing
// decode errors (the column is always written via MapToJSON, so a parse
// error here means corruption that the read path can do nothing about).
func DecodeStringMap(j datatypes.JSON) map[string]string {
	if len(j) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(j, &out); err != nil {
		return nil
	}
	return out
}

// IsUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505). Uses the typed pgconn error rather than
// substring-matching on the message so the check is locale-independent.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
