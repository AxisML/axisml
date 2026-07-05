// Package computeservice is a thin, typed wrapper over the generated
// compute-service client. The {namespace} path segment is the tenant scope
// (identifier). Identity is injected via the shared request editor; downstream
// problems map to Platform business errors.
package computeservice

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/reqedit"
)

const service = "compute"

// Clean-named aliases for the generated wire types used by callers.
type (
	MLRun         = gen.MLRun
	MLRunCreate   = gen.MLRunCreateRequest
	MLRunRole     = gen.MLRunRoleSpec
	MLRunBackend  = gen.MLRunBackendSpec
	MLRunTemplate = gen.MLRunPodTemplateSubset
	MLRunPolicy   = gen.MLRunRunPolicySpec
	MLRunPatch    = gen.MLRunPatchRequest
	EnvVar        = gen.Corev1EnvVar
	ResourceReqs  = gen.Corev1ResourceRequirements

	MLService        = gen.MLService
	MLServiceCreate  = gen.MLServiceCreateRequest
	MLServiceRole    = gen.MLServiceRoleSpec
	MLServiceBackend = gen.MLServiceBackend
	MLServiceTmpl    = gen.MLServicePodTemplate
	MLServiceRoute   = gen.MLServiceRoute
	MLServicePatch   = gen.MLServicePatchRequest
	ServicePort      = gen.MLServicePodPort

	TrafficPolicy   = gen.TrafficPolicy
	TrafficCreate   = gen.TrafficPolicyCreateRequest
	TrafficBackend  = gen.MLTrafficPolicyBackendMember
	TrafficEndpoint = gen.MLTrafficPolicyEndpoint
	TrafficPatch    = gen.TrafficPolicyPatchRequest
	WeightUpdate    = gen.TrafficPolicyWeightUpdate

	Pod   = gen.Pod
	Event = gen.Event
)

// Client wraps the generated compute client. A second client (stream) is used
// for log-follow / SSE endpoints, which must not carry an overall http.Client
// timeout (that deadline covers the whole body read and would sever a
// long-lived follow stream); their lifetime is governed by the request context.
type Client struct{ gen, stream *gen.ClientWithResponses }

// New builds a compute client for baseURL.
func New(baseURL string, timeout time.Duration) (*Client, error) {
	c, err := gen.NewClientWithResponses(baseURL,
		gen.WithHTTPClient(&http.Client{Timeout: timeout}),
		gen.WithRequestEditorFn(reqedit.Identity),
	)
	if err != nil {
		return nil, err
	}
	// Streaming client: bound connection setup and time-to-first-byte with
	// timeout, but no overall deadline — the stream runs until the caller's
	// context is cancelled.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = timeout
	sc, err := gen.NewClientWithResponses(baseURL,
		gen.WithHTTPClient(&http.Client{Transport: tr}),
		gen.WithRequestEditorFn(reqedit.Identity),
	)
	if err != nil {
		return nil, err
	}
	return &Client{gen: c, stream: sc}, nil
}

func listParams(labelSelector string) *gen.ListMLRunsParams {
	p := &gen.ListMLRunsParams{}
	if labelSelector != "" {
		p.LabelSelector = &labelSelector
	}
	return p
}

// ---- MLRun ----

// CreateMLRun creates a run.
func (c *Client) CreateMLRun(ctx context.Context, ns string, in MLRunCreate) (*MLRun, error) {
	res, err := c.gen.CreateMLRunWithResponse(ctx, ns, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// GetMLRun returns a run.
func (c *Client) GetMLRun(ctx context.Context, ns, name string) (*MLRun, error) {
	res, err := c.gen.GetMLRunWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListMLRuns lists runs filtered by labelSelector.
func (c *Client) ListMLRuns(ctx context.Context, ns, labelSelector string) ([]MLRun, error) {
	res, err := c.gen.ListMLRunsWithResponse(ctx, ns, listParams(labelSelector))
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// DeleteMLRun deletes a run.
func (c *Client) DeleteMLRun(ctx context.Context, ns, name string) error {
	res, err := c.gen.DeleteMLRunWithResponse(ctx, ns, name)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if ok2xx(res.HTTPResponse) {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// CancelMLRun requests cancellation.
func (c *Client) CancelMLRun(ctx context.Context, ns, name string) (*MLRun, error) {
	res, err := c.gen.CancelMLRunWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON202 != nil {
		return res.JSON202, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListMLRunPods lists a run's pods.
func (c *Client) ListMLRunPods(ctx context.Context, ns, name string) ([]Pod, error) {
	res, err := c.gen.ListMLRunPodsWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// ListMLRunEvents lists a run's events.
func (c *Client) ListMLRunEvents(ctx context.Context, ns, name string) ([]Event, error) {
	res, err := c.gen.ListMLRunEventsWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// ListMLRunPodEvents lists events for one pod of a run.
func (c *Client) ListMLRunPodEvents(ctx context.Context, ns, name, pod string) ([]Event, error) {
	res, err := c.gen.ListMLRunPodEventsWithResponse(ctx, ns, name, pod)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// LogOptions are the pod-log query knobs (container/tailLines/follow/previous).
type LogOptions struct {
	Container string
	TailLines *int
	Follow    bool
	Previous  bool
}

// StreamMLRunPodLogs streams a pod's logs. It returns the raw upstream response
// so the caller can pipe it through (one-shot text/plain, or SSE when
// follow=true); the caller MUST close resp.Body.
func (c *Client) StreamMLRunPodLogs(ctx context.Context, ns, name, pod string, opt LogOptions) (*http.Response, error) {
	p := &gen.GetMLRunPodLogsParams{}
	if opt.Container != "" {
		p.Container = &opt.Container
	}
	if opt.TailLines != nil {
		v := int32(*opt.TailLines)
		p.TailLines = &v
	}
	if opt.Follow {
		p.Follow = &opt.Follow
	}
	if opt.Previous {
		p.Previous = &opt.Previous
	}
	resp, err := c.gen.GetMLRunPodLogs(ctx, ns, name, pod, p)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	return checkStream(resp)
}

// ---- MLService ----

// CreateMLService creates a service / workspace / tensorboard.
func (c *Client) CreateMLService(ctx context.Context, ns string, in MLServiceCreate) (*MLService, error) {
	res, err := c.gen.CreateMLServiceWithResponse(ctx, ns, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// GetMLService returns a service.
func (c *Client) GetMLService(ctx context.Context, ns, name string) (*MLService, error) {
	res, err := c.gen.GetMLServiceWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListMLServices lists services filtered by labelSelector.
func (c *Client) ListMLServices(ctx context.Context, ns, labelSelector string) ([]MLService, error) {
	p := &gen.ListMLServicesParams{}
	if labelSelector != "" {
		p.LabelSelector = &labelSelector
	}
	res, err := c.gen.ListMLServicesWithResponse(ctx, ns, p)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// PatchMLService updates display metadata / labels / annotations (used for the
// last-replicas annotation in stop/start, §5.5).
func (c *Client) PatchMLService(ctx context.Context, ns, name string, in MLServicePatch) (*MLService, error) {
	res, err := c.gen.PatchMLServiceWithResponse(ctx, ns, name, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ScaleMLService sets the replica count.
func (c *Client) ScaleMLService(ctx context.Context, ns, name string, replicas int) (*MLService, error) {
	res, err := c.gen.ScaleMLServiceWithResponse(ctx, ns, name, gen.MLServiceScaleRequest{Replicas: int32(replicas)})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON202 != nil {
		return res.JSON202, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// DeleteMLService deletes a service. A workspace's durable volume is reclaimed
// by Platform via cluster-manager, not through this call.
func (c *Client) DeleteMLService(ctx context.Context, ns, name string) error {
	res, err := c.gen.DeleteMLServiceWithResponse(ctx, ns, name)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if ok2xx(res.HTTPResponse) {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListMLServicePods lists a service's pods.
func (c *Client) ListMLServicePods(ctx context.Context, ns, name string) ([]Pod, error) {
	res, err := c.gen.ListMLServicePodsWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// ListMLServiceEvents lists a service's events.
func (c *Client) ListMLServiceEvents(ctx context.Context, ns, name string) ([]Event, error) {
	res, err := c.gen.ListMLServiceEventsWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// ListMLServicePodEvents lists events for one pod of a service.
func (c *Client) ListMLServicePodEvents(ctx context.Context, ns, name, pod string) ([]Event, error) {
	res, err := c.gen.ListMLServicePodEventsWithResponse(ctx, ns, name, pod)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// StreamMLServicePodLogs streams an MLService pod's logs (see StreamMLRunPodLogs).
func (c *Client) StreamMLServicePodLogs(ctx context.Context, ns, name, pod string, opt LogOptions) (*http.Response, error) {
	p := &gen.GetMLServicePodLogsParams{}
	if opt.Container != "" {
		p.Container = &opt.Container
	}
	if opt.TailLines != nil {
		v := int32(*opt.TailLines)
		p.TailLines = &v
	}
	if opt.Follow {
		p.Follow = &opt.Follow
	}
	if opt.Previous {
		p.Previous = &opt.Previous
	}
	resp, err := c.gen.GetMLServicePodLogs(ctx, ns, name, pod, p)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	return checkStream(resp)
}

// checkStream returns the response for 2xx; otherwise reads the (bounded) error
// body and maps it, closing the response.
func checkStream(resp *http.Response) (*http.Response, error) {
	if ok2xx(resp) {
		return resp, nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	_ = resp.Body.Close()
	return nil, clienterr.FromResponse(service, resp, body)
}

// ---- TrafficPolicy ----

// CreateTrafficPolicy creates a traffic policy.
func (c *Client) CreateTrafficPolicy(ctx context.Context, ns string, in TrafficCreate) (*TrafficPolicy, error) {
	res, err := c.gen.CreateTrafficPolicyWithResponse(ctx, ns, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// GetTrafficPolicy returns a policy.
func (c *Client) GetTrafficPolicy(ctx context.Context, ns, name string) (*TrafficPolicy, error) {
	res, err := c.gen.GetTrafficPolicyWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// PatchTrafficPolicy updates display metadata / labels / annotations
// (no CR touch, no generation bump; weights go through split).
func (c *Client) PatchTrafficPolicy(ctx context.Context, ns, name string, in TrafficPatch) (*TrafficPolicy, error) {
	res, err := c.gen.PatchTrafficPolicyWithResponse(ctx, ns, name, in)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListTrafficPolicies lists policies filtered by labelSelector.
func (c *Client) ListTrafficPolicies(ctx context.Context, ns, labelSelector string) ([]TrafficPolicy, error) {
	p := &gen.ListTrafficPoliciesParams{}
	if labelSelector != "" {
		p.LabelSelector = &labelSelector
	}
	res, err := c.gen.ListTrafficPoliciesWithResponse(ctx, ns, p)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// DeleteTrafficPolicy deletes a policy (members are retained).
func (c *Client) DeleteTrafficPolicy(ctx context.Context, ns, name string) error {
	res, err := c.gen.DeleteTrafficPolicyWithResponse(ctx, ns, name)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if ok2xx(res.HTTPResponse) {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// SplitTrafficPolicy adjusts per-backend weights.
func (c *Client) SplitTrafficPolicy(ctx context.Context, ns, name string, backends []WeightUpdate) (*TrafficPolicy, error) {
	res, err := c.gen.SplitTrafficPolicyWithResponse(ctx, ns, name, gen.TrafficPolicySplitRequest{Backends: backends})
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON202 != nil {
		return res.JSON202, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// PromoteTrafficPolicy promotes the canary to stable.
func (c *Client) PromoteTrafficPolicy(ctx context.Context, ns, name string) (*TrafficPolicy, error) {
	res, err := c.gen.PromoteTrafficPolicyWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON202 != nil {
		return res.JSON202, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// RollbackTrafficPolicy rolls the canary back to the stable backend.
func (c *Client) RollbackTrafficPolicy(ctx context.Context, ns, name string) (*TrafficPolicy, error) {
	res, err := c.gen.RollbackTrafficPolicyWithResponse(ctx, ns, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON202 != nil {
		return res.JSON202, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

func ok2xx(resp *http.Response) bool {
	return resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300
}
