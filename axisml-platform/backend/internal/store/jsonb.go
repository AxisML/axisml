// Package store is Platform's GORM data-access layer: models and repositories
// for the durable tenant record, identity/authz/sessions, and the four
// name-level definition tables. It is the ONLY package that touches the DB.
package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StrMap is a string→string map persisted as a jsonb column.
type StrMap map[string]string

// Value implements driver.Valuer.
func (m StrMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements sql.Scanner.
func (m *StrMap) Scan(v any) error {
	if v == nil {
		*m = nil
		return nil
	}
	var b []byte
	switch t := v.(type) {
	case []byte:
		b = t
	case string:
		b = []byte(t)
	default:
		return fmt.Errorf("StrMap: unsupported scan type %T", v)
	}
	if len(b) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(b, m)
}

// JSONB is a raw JSON document persisted as a jsonb column. It round-trips
// opaque spec bodies without an intermediate Go shape.
type JSONB json.RawMessage

// Value implements driver.Valuer.
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan implements sql.Scanner.
func (j *JSONB) Scan(v any) error {
	if v == nil {
		*j = nil
		return nil
	}
	switch t := v.(type) {
	case []byte:
		cp := make([]byte, len(t))
		copy(cp, t)
		*j = cp
	case string:
		*j = []byte(t)
	default:
		return fmt.Errorf("JSONB: unsupported scan type %T", v)
	}
	return nil
}

// MarshalJSON renders the raw document (or {} when empty).
func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("{}"), nil
	}
	return j, nil
}

// UnmarshalJSON stores the raw document.
func (j *JSONB) UnmarshalJSON(b []byte) error {
	cp := make([]byte, len(b))
	copy(cp, b)
	*j = cp
	return nil
}
