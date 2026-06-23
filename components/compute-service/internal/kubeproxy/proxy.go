// Package kubeproxy renders the instance (Pod) projections that the Job and
// Service URL sub-trees expose: list instances, stream an instance log, list
// instance / resource events. The data now comes from the ComputeRuntime
// contract (so the same handlers serve any runtime form); this package only
// projects the returned Kubernetes types onto the stable REST DTOs and maps
// runtime errors onto HTTP codes.
package kubeproxy

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/axisml/axisml/components/compute-service/internal/server"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// MapErr translates a runtime error into the typed app error the HTTP error
// middleware understands. A Kubernetes NotFound (missing workload / instance)
// becomes CodeNotFound; anything else is surfaced as CodeUnavailable.
func MapErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := apperrors.As(err); ok {
		return err
	}
	if apierrors.IsNotFound(err) {
		return apperrors.Wrap(apperrors.CodeNotFound, "instance not found", err)
	}
	return apperrors.Wrap(apperrors.CodeUnavailable, "runtime call", err)
}

// WritePods renders an instance list as JSON, trimmed to a stable projection.
func WritePods(c *gin.Context, pods *corev1.PodList) {
	items := make([]server.Pod, 0, len(pods.Items))
	for i := range pods.Items {
		items = append(items, projectPod(&pods.Items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// WriteEvents renders an event list as JSON.
func WriteEvents(c *gin.Context, evs *eventsv1.EventList) {
	items := make([]server.Event, 0, len(evs.Items))
	for i := range evs.Items {
		items = append(items, projectEvent(&evs.Items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// PodLogQuery parses the /logs query params into the runtime's PodLogOptions
// plus the follow flag (which drives SSE framing).
func PodLogQuery(c *gin.Context) (*corev1.PodLogOptions, bool) {
	opts := &corev1.PodLogOptions{Container: c.Query("container")}
	if v := c.Query("tailLines"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			opts.TailLines = &n
		}
	}
	if v := c.Query("sinceSeconds"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			opts.SinceSeconds = &n
		}
	}
	follow := c.Query("follow") == "true"
	opts.Follow = follow
	opts.Previous = c.Query("previous") == "true"
	return opts, follow
}

// StreamLog opens an instance log stream via open and copies it to the client.
// follow=true switches to SSE framing with per-chunk flushing. A NotFound
// surfaces as an inline "410 Gone" marker (headers may already be on the wire),
// letting clients distinguish a GC'd instance from a transient failure.
func StreamLog(c *gin.Context, follow bool, open func() (io.ReadCloser, error)) {
	stream, err := open()
	if err != nil {
		if mapped := MapErr(err); mapped != nil {
			if e, ok := apperrors.As(mapped); ok && e.Code == apperrors.CodeNotFound {
				c.Header("Content-Type", "text/plain; charset=utf-8")
				c.Status(http.StatusGone)
				_, _ = c.Writer.WriteString("[instance gone: 410 Gone]\n")
				return
			}
			_ = c.Error(mapped)
		}
		return
	}
	defer func() { _ = stream.Close() }()

	var flusher http.Flusher
	if follow {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		if f, ok := c.Writer.(http.Flusher); ok {
			flusher = f
		}
	} else {
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	c.Status(http.StatusOK)

	if flusher == nil {
		_, _ = io.Copy(c.Writer, stream)
		return
	}
	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			if _, wErr := c.Writer.Write(buf[:n]); wErr != nil {
				return
			}
			flusher.Flush()
		}
		if readErr == io.EOF {
			return
		}
		if readErr != nil {
			return
		}
	}
}

// AbortIfMissingNamespace short-circuits with 400 when the URL :namespace
// param is empty (defensive; gin path matching shouldn't normally allow it).
func AbortIfMissingNamespace(g *gin.Context) bool {
	if g.Param("namespace") == "" {
		_ = g.Error(apperrors.New(apperrors.CodeValidation, "namespace path parameter is required"))
		return true
	}
	return false
}

func projectPod(p *corev1.Pod) server.Pod {
	return server.Pod{
		Name:      p.Name,
		Namespace: p.Namespace,
		Phase:     string(p.Status.Phase),
		NodeName:  p.Spec.NodeName,
		Labels:    p.Labels,
	}
}

func projectEvent(e *eventsv1.Event) server.Event {
	return server.Event{
		Reason:    e.Reason,
		Note:      e.Note,
		Type:      e.Type,
		Object:    e.Regarding.Kind + "/" + e.Regarding.Name,
		Reporter:  e.ReportingController,
		EventTime: ptrTime(e.EventTime),
	}
}

func ptrTime(t metav1.MicroTime) *metav1.Time {
	if t.IsZero() {
		return nil
	}
	mt := metav1.NewTime(t.Time)
	return &mt
}
