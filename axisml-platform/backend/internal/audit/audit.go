// Package audit records Platform-side mutations into a durable audit store,
// powering the dashboard recent-activity feed. Recording is a cross-cutting
// gin middleware: after any successful mutating /api/v1 request it derives the
// subject (kind, name), the action, the acting user and the tenant scope from
// the resolved route + identity, and appends one audit event. Failures are
// swallowed — auditing must never break the request it observes.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
)

const captureLimit = 64 << 10 // cap buffered response bytes for create-name extraction

// kindByPath maps the leading /api/v1/<segment> to an audit subject kind.
var kindByPath = map[string]string{
	"jobs":            "job",
	"experiments":     "experiment",
	"mlservices":      "service",
	"workspaces":      "workspace",
	"trafficpolicies": "trafficpolicy",
	"tenants":         "tenant",
	"models":          "model",
	"images":          "image",
	"resourcepools":   "resourcepool",
	"datavolumes":     "datavolume",
}

// actionBySuffix maps a trailing POST sub-action segment to an action verb.
var actionBySuffix = map[string]string{
	"cancel":   "canceled",
	"scale":    "scaled",
	"stop":     "stopped",
	"start":    "started",
	"split":    "split",
	"promote":  "promoted",
	"rollback": "rolled-back",
	"suspend":  "suspended",
	"resume":   "resumed",
}

// Recorder persists audit events derived from HTTP requests.
type Recorder struct {
	repo *store.AuditRepo
	log  *slog.Logger
}

// NewRecorder constructs a Recorder over the audit store.
func NewRecorder(repo *store.AuditRepo, log *slog.Logger) *Recorder {
	return &Recorder{repo: repo, log: log}
}

// Middleware records one audit event per successful mutating request. It is
// mounted on the /api/v1 group so it observes the resolved route params and the
// identity set by each module's auth middleware.
func (r *Recorder) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isMutating(c.Request.Method) {
			c.Next()
			return
		}
		cap := &respCapture{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = cap
		c.Next()

		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}
		ev := r.derive(c, cap.buf.Bytes())
		if ev == nil {
			return
		}
		// Detach from the request context (already done) so a client disconnect
		// does not drop the record; keep it synchronous so the row is durable
		// before the connection is reused.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := r.repo.Insert(ctx, ev); err != nil {
			r.log.Debug("audit insert failed", "err", err, "kind", ev.Kind, "action", ev.Action, "name", ev.Name)
		}
	}
}

// derive maps a completed request to an audit event, or nil when the request is
// not an auditable business mutation.
func (r *Recorder) derive(c *gin.Context, body []byte) *store.AuditEvent {
	segs := nonEmpty(strings.Split(strings.TrimPrefix(c.Request.URL.Path, "/api/v1/"), "/"))
	if len(segs) == 0 {
		return nil
	}
	kind, ok := kindByPath[segs[0]]
	if !ok {
		return nil
	}
	last := segs[len(segs)-1]

	// Refine the subject kind for known sub-resources.
	switch {
	case c.Param("run") != "" || last == "runs":
		kind = "run"
	case contains(segs, "quotas"):
		kind = "quota"
	case contains(segs, "members"):
		kind = "member"
	}

	var action string
	switch c.Request.Method {
	case http.MethodPost:
		if v, ok := actionBySuffix[last]; ok {
			action = v
		} else {
			action = "created"
		}
	case http.MethodPatch, http.MethodPut:
		action = "updated"
	case http.MethodDelete:
		action = "deleted"
	default:
		return nil
	}

	name := firstParam(c, "run", "name", "identifier", "pool", "version")
	if name == "" {
		name = fieldFromJSON(body, "name", "identifier")
	}
	if name == "" {
		return nil
	}

	tenant := auth.ActiveTenant(c)
	if tenant == "" {
		tenant = c.Param("identifier")
	}
	if tenant == "" {
		tenant = fieldFromJSON(body, "tenantName", "namespace", "identifier")
	}

	actor := ""
	if id := auth.Current(c); id != nil {
		actor = id.Username
	}

	return &store.AuditEvent{Tenant: tenant, Kind: kind, Name: name, Action: action, Actor: actor}
}

// respCapture tees the response body into a bounded buffer so create handlers'
// returned entity name can be recovered when it is not a path parameter.
type respCapture struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *respCapture) Write(b []byte) (int, error) {
	if w.buf.Len() < captureLimit {
		w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *respCapture) WriteString(s string) (int, error) {
	if w.buf.Len() < captureLimit {
		w.buf.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

func firstParam(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := c.Param(k); v != "" {
			return v
		}
	}
	return ""
}

func fieldFromJSON(body []byte, keys ...string) string {
	if len(body) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func nonEmpty(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}
