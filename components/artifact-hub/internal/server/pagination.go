package server

import (
	"encoding/base64"
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
)

// Pagination is the parsed list-query control block. Per the design yaml
// list endpoints use the K8s-style opaque `continue` token; internally
// the token encodes the next-page offset (base64'd to keep clients from
// parsing it).
type Pagination struct {
	Limit  int
	Offset int
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

// ParsePagination reads `?limit` / `?continue` (with `?offset=N` as a
// back-compat alias) from the query string.
func ParsePagination(c *gin.Context) (Pagination, error) {
	p := Pagination{Limit: defaultLimit}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return p, apperrors.New(apperrors.CodeValidation, "limit must be a positive integer")
		}
		if n > maxLimit {
			n = maxLimit
		}
		p.Limit = n
	}
	if v := c.Query("continue"); v != "" {
		n, err := decodeContinue(v)
		if err != nil || n < 0 {
			return p, apperrors.New(apperrors.CodeValidation, "continue token is invalid")
		}
		p.Offset = n
	} else if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return p, apperrors.New(apperrors.CodeValidation, "offset must be a non-negative integer")
		}
		p.Offset = n
	}
	return p, nil
}

// EncodeContinue produces the next-page token to surface alongside the
// list response. Returns "" when there is no next page.
func EncodeContinue(offset, count int, total int64) string {
	if int64(offset+count) >= total {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset + count)))
}

func decodeContinue(raw string) (int, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(b))
}
