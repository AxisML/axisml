//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	poolv1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	mljobv1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// suite is the shared, process-wide harness wired up in TestMain.
type suite struct {
	cfg envConfig

	scheme *runtime.Scheme
	k8s    client.Client

	clusterManager *httpClient
	computeService *httpClient
	artifactHub    *httpClient

	// forwards are torn down at the end of the run.
	forwards []*portForward
}

// h is the global harness handle. TestMain populates it before any test runs.
var h *suite

// buildScheme registers every typed API the suite reads or writes. Koordinator
// ElasticQuota and gateway-api HTTPRoute are intentionally accessed as
// unstructured objects (see quotaObj / httpRouteObj) to avoid pulling those
// heavyweight modules into the e2e go.mod.
func buildScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(tenantv1.AddToScheme(s))
	utilruntime.Must(poolv1.AddToScheme(s))
	utilruntime.Must(mljobv1.AddToScheme(s))
	utilruntime.Must(mlservicev1.AddToScheme(s))
	utilruntime.Must(mltpv1.AddToScheme(s))
	return s
}

// newSuite builds the K8s client from the ambient kubeconfig and starts the
// port-forwards. It does NOT create the shared tenant — that happens in
// TestMain after a readiness gate so failures are reported clearly.
func newSuite() (*suite, error) {
	cfg := loadConfig()
	// Give the shared tenant a per-run-unique name unless explicitly pinned.
	// compute-service soft-deletes tenants, so reusing a fixed name across runs
	// collides (409 on re-create, leaving no CR/quota). A unique name sidesteps
	// that; cleanup hard-removes the CR + namespace via the admin client.
	if v := os.Getenv("E2E_SHARED_TENANT"); v != "" {
		cfg.SharedTenant = v
	} else {
		cfg.SharedTenant = fmt.Sprintf("e2e-%d", time.Now().Unix()%1000000)
	}
	cfg.SharedNamespace = envOr("E2E_SHARED_NAMESPACE", cfg.SharedTenant)
	s := &suite{cfg: cfg, scheme: buildScheme()}

	restCfg, err := ctrlconfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig (is the axisml cluster up?): %w", err)
	}
	cl, err := client.New(restCfg, client.Options{Scheme: s.scheme})
	if err != nil {
		return nil, fmt.Errorf("build k8s client: %w", err)
	}
	s.k8s = cl

	cm, err := s.forward(cfg.SystemNamespace, cfg.ClusterManagerSvc, cfg.ClusterManagerPort)
	if err != nil {
		return nil, fmt.Errorf("port-forward cluster-manager: %w", err)
	}
	cs, err := s.forward(cfg.SystemNamespace, cfg.ComputeServiceSvc, cfg.ComputeServicePort)
	if err != nil {
		return nil, fmt.Errorf("port-forward compute-service: %w", err)
	}
	ah, err := s.forward(cfg.SystemNamespace, cfg.ArtifactHubSvc, cfg.ArtifactHubPort)
	if err != nil {
		return nil, fmt.Errorf("port-forward artifact-hub: %w", err)
	}

	s.clusterManager = newHTTPClient(cm.localURL(), cfg.User)
	s.computeService = newHTTPClient(cs.localURL(), cfg.User)
	s.artifactHub = newHTTPClient(ah.localURL(), cfg.User)
	return s, nil
}

func (s *suite) forward(ns, svc string, remotePort int) (*portForward, error) {
	pf, err := startPortForward(ns, svc, remotePort)
	if err != nil {
		return nil, err
	}
	s.forwards = append(s.forwards, pf)
	return pf, nil
}

func (s *suite) close() {
	for _, pf := range s.forwards {
		pf.Stop()
	}
}

// ---------------------------------------------------------------------------
// kubectl port-forward
// ---------------------------------------------------------------------------

var forwardingRe = regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)`)

// portForward wraps a `kubectl port-forward svc/<name> :<remote>` subprocess.
// We let kubectl pick the local port (the ":remote" form) and parse it from
// stdout — reimplementing the SPDY upgrade in-process is brittle, and the e2e
// host already has kubectl + a working kubeconfig by assumption.
type portForward struct {
	svc       string
	localPort int
	cmd       *exec.Cmd
}

func startPortForward(ns, svc string, remotePort int) (*portForward, error) {
	args := []string{"port-forward", "-n", ns, "svc/" + svc, fmt.Sprintf(":%d", remotePort)}
	cmd := exec.Command("kubectl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout // fold stderr in so failures surface in the scan
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start kubectl port-forward: %w", err)
	}

	pf := &portForward{svc: svc, cmd: cmd}
	portCh := make(chan int, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if m := forwardingRe.FindStringSubmatch(sc.Text()); m != nil {
				var p int
				fmt.Sscanf(m[1], "%d", &p)
				select {
				case portCh <- p:
				default:
				}
			}
		}
	}()

	select {
	case p := <-portCh:
		pf.localPort = p
		// Give the listener a moment to accept connections.
		if err := waitListen(p, 10*time.Second); err != nil {
			pf.Stop()
			return nil, err
		}
		return pf, nil
	case <-time.After(30 * time.Second):
		pf.Stop()
		return nil, fmt.Errorf("timed out waiting for kubectl port-forward to svc/%s in %s", svc, ns)
	}
}

func (pf *portForward) localURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", pf.localPort)
}

func (pf *portForward) Stop() {
	if pf.cmd != nil && pf.cmd.Process != nil {
		_ = pf.cmd.Process.Kill()
		_ = pf.cmd.Wait()
	}
}

func waitListen(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("nothing listening on %s after %s", addr, timeout)
}

// ---------------------------------------------------------------------------
// HTTP client
// ---------------------------------------------------------------------------

const headerUser = "X-Axisml-User"

type httpClient struct {
	baseURL string
	user    string
	c       *http.Client
}

func newHTTPClient(baseURL, user string) *httpClient {
	return &httpClient{baseURL: baseURL, user: user, c: &http.Client{Timeout: 30 * time.Second}}
}

// resp is the decoded result of an HTTP call.
type resp struct {
	status int
	body   []byte
}

func (r resp) is2xx() bool { return r.status >= 200 && r.status < 300 }
func (r resp) is4xx() bool { return r.status >= 400 && r.status < 500 }

// decode unmarshals the JSON body into out. Callers use it only on 2xx.
func (r resp) decode(out any) error {
	if out == nil {
		return nil
	}
	return json.Unmarshal(r.body, out)
}

// do issues a request with the identity header. body may be nil.
func (hc *httpClient) do(ctx context.Context, method, path string, body any) (resp, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return resp{}, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, hc.baseURL+path, rdr)
	if err != nil {
		return resp{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if hc.user != "" {
		req.Header.Set(headerUser, hc.user)
	}
	httpResp, err := hc.c.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer httpResp.Body.Close()
	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp{}, err
	}
	return resp{status: httpResp.StatusCode, body: b}, nil
}

// doNoAuth is like do but omits the identity header (for 401 negative tests).
func (hc *httpClient) doNoAuth(ctx context.Context, method, path string, body any) (resp, error) {
	bare := &httpClient{baseURL: hc.baseURL, user: "", c: hc.c}
	return bare.do(ctx, method, path, body)
}

// mustDo fails the test on transport error and returns the response.
func (hc *httpClient) mustDo(t *testing.T, ctx context.Context, method, path string, body any) resp {
	t.Helper()
	r, err := hc.do(ctx, method, path, body)
	if err != nil {
		t.Fatalf("%s %s: transport error: %v", method, path, err)
	}
	return r
}

// ---------------------------------------------------------------------------
// Unstructured accessors for external CRDs (ElasticQuota, HTTPRoute)
// ---------------------------------------------------------------------------

func quotaObj() *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "scheduling.sigs.k8s.io",
		Version: "v1alpha1",
		Kind:    "ElasticQuota",
	})
	return o
}

func httpRouteObj() *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	return o
}

// ---------------------------------------------------------------------------
// Polling helpers (e2e budgets; thin wrappers over the stdlib)
// ---------------------------------------------------------------------------

// eventually polls fn until it returns nil or the timeout elapses, using the
// suite's poll interval. Mirrors testutil.Eventually but keeps the e2e budgets.
func eventually(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		if last = fn(); last == nil {
			return
		}
		time.Sleep(h.cfg.PollInterval)
	}
	if last == nil {
		last = fmt.Errorf("condition not met")
	}
	t.Fatalf("eventually: timed out after %s: %v", timeout, last)
}

// consistently asserts fn returns nil on every poll across the whole window
// (the inverse of eventually). Used for "must STAY in state X" checks where a
// single immediate read could pass before the system has had a chance to act.
func consistently(t *testing.T, window time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		if err := fn(); err != nil {
			t.Fatalf("consistently: condition violated within %s: %v", window, err)
		}
		time.Sleep(h.cfg.PollInterval)
	}
}

func (s *suite) get(ctx context.Context, ns, name string, obj client.Object) error {
	return s.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, obj)
}

// namespaceExists reports whether a namespace exists in the cluster.
func (s *suite) namespaceExists(ctx context.Context, name string) error {
	var ns corev1.Namespace
	if err := s.k8s.Get(ctx, client.ObjectKey{Name: name}, &ns); err != nil {
		return err
	}
	return nil
}

func isNotFound(err error) bool { return apierrors.IsNotFound(err) }

// objMeta is a tiny helper to build ObjectMeta inline.
func objMeta(ns, name string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: ns, Name: name, Labels: labels}
}
