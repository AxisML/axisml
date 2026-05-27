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

// ListPodsByLabel returns Pods in `namespace` whose metadata.labels match
// the given key=value (typically `axisml.io/job-id=<uuid>` or
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

// EventsForInvolved returns events whose .involvedObject matches the
// (kind, name) under namespace.
func (c *Client) EventsForInvolved(ctx context.Context, namespace, kind, name string) ([]eventsv1.Event, error) {
	var evList eventsv1.EventList
	if err := c.ctrl.List(ctx, &evList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnavailable, "list events", err)
	}
	out := make([]eventsv1.Event, 0, len(evList.Items))
	for _, e := range evList.Items {
		if e.Regarding.Kind == kind && e.Regarding.Name == name {
			out = append(out, e)
		}
	}
	return out, nil
}

// StreamPodLog writes the pod's container log to w. tailLines / follow
// pass through to kube-apiserver.
func (c *Client) StreamPodLog(ctx context.Context, namespace, pod, container string, tailLines int64, follow bool, w io.Writer) error {
	opts := &corev1.PodLogOptions{Container: container, Follow: follow}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
	req := c.clients.CoreV1().Pods(namespace).GetLogs(pod, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return apperrors.Newf(apperrors.CodeNotFound, "pod %s/%s not found", namespace, pod)
		}
		return apperrors.Wrap(apperrors.CodeUnavailable, "open pod log stream", err)
	}
	defer stream.Close()
	_, err = io.Copy(w, stream)
	return err
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
	items := make([]PodView, 0, len(pods))
	for i := range pods {
		items = append(items, projectPod(&pods[i]))
	}
	g.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// PodLog streams text/plain.
func (c *Client) PodLog(g *gin.Context, namespace, pod string) {
	container := g.Query("container")
	tail := int64(0)
	if v := g.Query("tailLines"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			tail = n
		}
	}
	follow := g.Query("follow") == "true"

	g.Status(http.StatusOK)
	g.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := c.StreamPodLog(g.Request.Context(), namespace, pod, container, tail, follow, g.Writer); err != nil {
		_ = g.Error(err)
	}
}

// EventsByInvolved renders events as JSON.
func (c *Client) EventsByInvolved(g *gin.Context, namespace, kind, name string) {
	evs, err := c.EventsForInvolved(g.Request.Context(), namespace, kind, name)
	if err != nil {
		_ = g.Error(err)
		return
	}
	items := make([]EventView, 0, len(evs))
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

// PodView is the stable projection returned to clients.
type PodView struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Phase     string            `json:"phase"`
	NodeName  string            `json:"nodeName,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

func projectPod(p *corev1.Pod) PodView {
	return PodView{
		Name:      p.Name,
		Namespace: p.Namespace,
		Phase:     string(p.Status.Phase),
		NodeName:  p.Spec.NodeName,
		Labels:    p.Labels,
	}
}

// EventView projects an events.k8s.io/v1 Event down to its useful fields.
type EventView struct {
	Reason    string       `json:"reason"`
	Note      string       `json:"note,omitempty"`
	Type      string       `json:"type"`
	Object    string       `json:"object"` // "<kind>/<name>"
	Reporter  string       `json:"reportingController,omitempty"`
	EventTime *metav1.Time `json:"eventTime,omitempty"`
}

func projectEvent(e *eventsv1.Event) EventView {
	return EventView{
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
