package server

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// Page holds normalised pagination inputs parsed from query params.
type Page struct {
	Limit    int
	Continue string
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

// ParsePage reads ?limit= and ?continue= with the contract defaults
// (default 50, max 200).
func ParsePage(c *gin.Context) Page {
	limit := defaultLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return Page{Limit: limit, Continue: c.Query("continue")}
}

// Offset converts an opaque continue token (a numeric offset) into an int.
func (p Page) Offset() int { return OffsetFromToken(p.Continue) }

// OffsetFromToken parses a numeric continue token (0 when empty/invalid).
func OffsetFromToken(tok string) int {
	if tok == "" {
		return 0
	}
	n, err := strconv.Atoi(tok)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// NextContinue returns the continue token for the next page, or "" when the
// returned count is short of the page limit (last page).
func NextContinue(offset, limit, returned int) string {
	if returned < limit {
		return ""
	}
	return strconv.Itoa(offset + returned)
}

// Fail forwards err to the ErrorHandler middleware (rendered as problem+json).
func Fail(c *gin.Context, err error) { _ = c.Error(err); c.Abort() }

// Forbidden is a 403 business error with the stable "forbidden" code.
func Forbidden() error {
	return apperrors.New(apperrors.ClassForbidden, "forbidden").WithReason("forbidden")
}

// NotFound is a 404 business error with the stable "not-found" code.
func NotFound(msg string) error {
	return apperrors.New(apperrors.ClassNotFound, msg).WithReason("not-found")
}

// Conflict is a 409 business error with the given stable reason code.
func Conflict(reason, msg string) error {
	return apperrors.New(apperrors.ClassConflict, msg).WithReason(reason)
}

// ActiveTenantRequired is the 400 returned when a list/name-addressed endpoint
// needs an X-Axisml-Tenant header that the caller did not supply.
func ActiveTenantRequired() error {
	return apperrors.New(apperrors.ClassValidation, "active tenant required").WithReason("active-tenant-required")
}

// MetricsUnavailable is the 502 returned when the compute-side metrics backend
// (Prometheus) is not configured. Platform proxies metrics to System (compute /
// cluster-manager) and must not query Prometheus directly.
func MetricsUnavailable() error {
	return apperrors.New(apperrors.ClassUpstream, "workload metrics are not yet available").WithReason("metrics-unavailable")
}

// OptStr returns a pointer to s, or nil when s is empty — for optional query
// params threaded to the typed System clients.
func OptStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
