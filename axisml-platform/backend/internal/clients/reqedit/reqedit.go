// Package reqedit provides the shared request editor that injects the
// X-Axisml-User identity header (backend.md §5.1, §6) into every outbound
// System-layer call, reading the username off the request context.
package reqedit

import (
	"context"
	"net/http"

	"github.com/axisml/axisml/components/platform/internal/auth"
)

// Identity sets X-Axisml-User from the context (no-op when absent).
func Identity(ctx context.Context, req *http.Request) error {
	if u := auth.UsernameFromContext(ctx); u != "" {
		req.Header.Set(auth.HeaderUser, u)
	}
	return nil
}
