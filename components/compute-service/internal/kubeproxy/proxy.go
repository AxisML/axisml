// Package kubeproxy hosts the kube-apiserver transit endpoints for
// compute-service: list pods by label, list events by involvedObject,
// stream pod logs. Design §4.3 / §4.4 places these on the Job and
// Service URL sub-trees; both handlers share this single client.
package kubeproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml/axisml/components/compute-service/internal/server"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// Client wraps the two access modes needed by the transit endpoints:
//   - controller-runtime client (typed, cached lists)
//   - clientset (untyped sub-resources like /pods/{name}/log streaming)
type Client struct {
	ctrl    client.Client
	clients kubernetes.Interface
}

// New builds a Client from a rest.Config.
func New(cfg *rest.Config, ctrl client.Client) (*Client, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubeproxy: build clientset: %w", err)
	}
	return &Client{ctrl: ctrl, clients: cs}, nil
}

// VerifyPodHasLabel fetches the named Pod and confirms it carries the
// expected label key=value. Returns CodeNotFound if the pod is missing or
// CodePermissionDenied if the pod exists but is not tagged with the
// caller's job/service id — preventing the URL `:namespace/:pod` from
// leaking pods belonging to other rows in the same K8s namespace.
func (c *Client) VerifyPodHasLabel(ctx context.Context, namespace, pod, labelKey, labelValue string) error {
	var p corev1.Pod
	if err := c.ctrl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: pod}, &p); err != nil {
		if apierrors.IsNotFound(err) {
			return apperrors.Newf(apperrors.CodeNotFound, "pod %s/%s not found", namespace, pod)
		}
		return apperrors.Wrap(apperrors.CodeUnavailable, "get pod", err)
	}
	if p.Labels[labelKey] != labelValue {
		return apperrors.Newf(apperrors.CodeForbidden,
			"pod %s/%s does not belong to this resource", namespace, pod)
	}
	return nil
}

// ListPodsByLabel returns Pods in `namespace` whose metadata.labels match
// the given key=value (typically `axisml.io/run-id=<uuid>` or
// `axisml.io/service-id=<uuid>`).
func (c *Client) ListPodsByLabel(ctx context.Context, namespace, labelKey, labelValue string) ([]corev1.Pod, error) {
	var pods corev1.PodList
	sel := labels.SelectorFromSet(labels.Set{labelKey: labelValue})
	if err := c.ctrl.List(ctx, &pods, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: sel,
	}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnavailable, "list pods", err)
	}
	return pods.Items, nil
}

// EventsForInvolved returns events whose .regarding (events.k8s.io/v1) or
// .involvedObject (core/v1) matches one of the (kind, name) targets under
// namespace. compute-service.md §4.3 / §4.4 specify that a job's /events
// endpoint includes MLRun + PodGroup events; a service's /events covers
// MLService + Deployment + StatefulSet + HTTPRoute. Callers pass the
// list as multiple `match` entries.
type EventTarget struct {
	Kind string
	Name string
}

func (c *Client) EventsForInvolved(ctx context.Context, namespace string, targets ...EventTarget) ([]eventsv1.Event, error) {
	var evList eventsv1.EventList
	if err := c.ctrl.List(ctx, &evList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnavailable, "list events", err)
	}
	out := make([]eventsv1.Event, 0, len(evList.Items))
	for _, e := range evList.Items {
		for _, t := range targets {
			if e.Regarding.Kind == t.Kind && e.Regarding.Name == t.Name {
				out = append(out, e)
				break
			}
		}
	}
	return out, nil
}

// PodLogOptions carries the query parameters the /logs endpoint forwards
// to kube-apiserver. Mirrors the design yaml's subset.
type PodLogOptions struct {
	Container    string
	Follow       bool
	Previous     bool
	TailLines    int64
	SinceSeconds int64
}

// StreamPodLog writes the pod's container log to w. NotFound is surfaced
// as CodeNotFound (the handler maps it to 410 Gone when a pod that was
// recently observed has been GC'd). When flusher is non-nil, each chunk
// gets flushed immediately (SSE / follow contract).
func (c *Client) StreamPodLog(ctx context.Context, namespace, pod string, opts PodLogOptions, w io.Writer, flusher http.Flusher) error {
	apiOpts := &corev1.PodLogOptions{
		Container: opts.Container,
		Follow:    opts.Follow,
		Previous:  opts.Previous,
	}
	if opts.TailLines > 0 {
		apiOpts.TailLines = &opts.TailLines
	}
	if opts.SinceSeconds > 0 {
		apiOpts.SinceSeconds = &opts.SinceSeconds
	}
	req := c.clients.CoreV1().Pods(namespace).GetLogs(pod, apiOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return apperrors.Newf(apperrors.CodeNotFound, "pod %s/%s not found", namespace, pod)
		}
		return apperrors.Wrap(apperrors.CodeUnavailable, "open pod log stream", err)
	}
	defer func() { _ = stream.Close() }()
	if flusher == nil {
		_, err = io.Copy(w, stream)
		return err
	}
	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			if _, wErr := w.Write(buf[:n]); wErr != nil {
				return wErr
			}
			flusher.Flush()
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

// ----------------------------------------------------------------------
// gin-handler convenience wrappers

// PodsByLabel renders the list as JSON; trims the Pod down to a stable
// projection (avoid leaking apiserver bytes blindly).
func (c *Client) PodsByLabel(g *gin.Context, namespace, labelKey, labelValue string) {
	pods, err := c.ListPodsByLabel(g.Request.Context(), namespace, labelKey, labelValue)
	if err != nil {
		_ = g.Error(err)
		return
	}
	items := make([]server.Pod, 0, len(pods))
	for i := range pods {
		items = append(items, projectPod(&pods[i]))
	}
	g.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// PodLog streams the log as text/plain (or as an SSE stream when ?follow=true).
// Maps NotFound to HTTP 410 Gone, which lets clients distinguish a GC'd
// pod from a transient kube-apiserver lookup failure.
func (c *Client) PodLog(g *gin.Context, namespace, pod string) {
	opts := PodLogOptions{Container: g.Query("container")}
	if v := g.Query("tailLines"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			opts.TailLines = n
		}
	}
	if v := g.Query("sinceSeconds"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			opts.SinceSeconds = n
		}
	}
	opts.Follow = g.Query("follow") == "true"
	opts.Previous = g.Query("previous") == "true"

	var flusher http.Flusher
	if opts.Follow {
		g.Writer.Header().Set("Content-Type", "text/event-stream")
		g.Writer.Header().Set("Cache-Control", "no-cache")
		g.Writer.Header().Set("Connection", "keep-alive")
		// Grab the underlying ResponseWriter's Flusher so each chunk
		// hits the wire immediately (SSE contract).
		if f, ok := g.Writer.(http.Flusher); ok {
			flusher = f
		}
	} else {
		g.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	g.Status(http.StatusOK)

	if err := c.StreamPodLog(g.Request.Context(), namespace, pod, opts, g.Writer, flusher); err != nil {
		// Distinguish "pod GC'd / never existed" from transient failures.
		if e, ok := apperrors.As(err); ok && e.Code == apperrors.CodeNotFound {
			// Headers may already be flushed; the best signal we can give a
			// client now is to write a marker line and return.
			_, _ = g.Writer.WriteString("\n[pod gone: 410 Gone]\n")
			return
		}
		_ = g.Error(err)
	}
}

// EventsByInvolved renders events targeting any of the supplied (kind, name)
// pairs as JSON. Used by both /jobs/{job}/events (MLRun + PodGroup) and
// /services/{svc}/events (MLService + Deployment + StatefulSet + HTTPRoute).
func (c *Client) EventsByInvolved(g *gin.Context, namespace string, targets ...EventTarget) {
	evs, err := c.EventsForInvolved(g.Request.Context(), namespace, targets...)
	if err != nil {
		_ = g.Error(err)
		return
	}
	items := make([]server.Event, 0, len(evs))
	for i := range evs {
		items = append(items, projectEvent(&evs[i]))
	}
	g.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
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
