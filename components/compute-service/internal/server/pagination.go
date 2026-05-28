package server

import (
	"encoding/base64"
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// Pagination is the parsed list-query control block. Per the design yaml
// we use a K8s-style opaque `continue` token instead of a numeric offset
// so list endpoints behave like server-paginated lists in K8s API.
//
// Internally the continue token encodes the next-page offset (base64'd
// to keep clients from making assumptions about its structure).
type Pagination struct {
	Limit  int
	Offset int
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

// ParsePagination reads `?limit` / `?continue` from the query string.
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
