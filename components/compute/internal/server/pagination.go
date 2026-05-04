package server

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Pagination is the parsed list-query control block.
type Pagination struct {
	Limit  int
	Offset int
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

// ParsePagination reads ?limit / ?offset from the query string.
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
	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return p, apperrors.New(apperrors.CodeValidation, "offset must be a non-negative integer")
		}
		p.Offset = n
	}
	return p, nil
}
