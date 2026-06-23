// Package clienterr maps downstream HTTP responses from the generated System-
// layer clients into Platform business errors (backend.md §5.1): 4xx pass
// through with the downstream code; 5xx and transport failures wrap as
// upstream-failure tagged with the service name.
package clienterr

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

// Transport wraps a connection-level failure (the generated call returned err).
func Transport(service string, err error) error {
	return apperrors.Wrap(apperrors.ClassUpstream, service+": request failed", err).
		WithReason("upstream-failure").
		WithDetails(map[string]any{"service": service})
}

// FromResponse converts a non-2xx downstream response into a business error.
// resp may be nil (defensive); body is the raw response body.
func FromResponse(service string, resp *http.Response, body []byte) error {
	status := http.StatusBadGateway
	if resp != nil {
		status = resp.StatusCode
	}
	var p problem
	_ = json.Unmarshal(body, &p)
	// Only trust the parsed problem fields for the user-facing message; never
	// surface a raw non-problem body (e.g. a gateway HTML error page) verbatim.
	msg := firstNonEmpty(p.Detail, p.Title)

	// 5xx — and any 401/403, since the System layer does no user-facing authz
	// (a rejection there means the Platform↔System trust/network failed, not the
	// caller's session) — map to an upstream failure tagged with the service.
	if status >= 500 || status == http.StatusUnauthorized || status == http.StatusForbidden {
		return apperrors.Newf(apperrors.ClassUpstream, "%s: %s", service, msg).
			WithReason("upstream-failure").
			WithDetails(map[string]any{"service": service, "status": status})
	}
	e := apperrors.New(classForStatus(status), msg)
	if p.Code != "" {
		e = e.WithReason(p.Code)
	}
	return e
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "downstream error"
}

func classForStatus(status int) apperrors.Class {
	switch status {
	case http.StatusBadRequest:
		return apperrors.ClassValidation
	case http.StatusUnauthorized:
		return apperrors.ClassUnauthorized
	case http.StatusForbidden:
		return apperrors.ClassForbidden
	case http.StatusNotFound:
		return apperrors.ClassNotFound
	case http.StatusConflict:
		return apperrors.ClassConflict
	case http.StatusGone:
		return apperrors.ClassGone
	case http.StatusUnprocessableEntity:
		return apperrors.ClassUnprocessable
	default:
		return apperrors.ClassValidation
	}
}
