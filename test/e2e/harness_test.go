//go:build (e2e || standard) && !lite

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
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
	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// suite is the shared, process-wide harness wired up in TestMain.
type suite struct {
	cfg envConfig

	scheme *runtime.Scheme
	k8s    client.Client

	// Typed, OpenAPI-generated clients for the three System HTTP components. They
	// reach the in-cluster Services over the port-forwards started in newSuite and
	// carry the suite's identity header via a client-wide request editor.
	clusterManager *clustermanager.ClientWithResponses
	computeService *computeservice.ClientWithResponses
	artifactHub    *artifacthub.ClientWithResponses

	// forwards are torn down at the end of the run.
	forwards []*portForward
}

// h is the global harness handle. TestMain populates it before any test runs.
var h *suite

// buildScheme registers every typed API the suite reads or writes. The
// gateway-api HTTPRoute is intentionally accessed as an unstructured object
// (see httpRouteObj) to avoid pulling that heavyweight module into the e2e
// go.mod.
func buildScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(tenantv1.AddToScheme(s))
	utilruntime.Must(poolv1.AddToScheme(s))
	utilruntime.Must(mlrunv1.AddToScheme(s))
	utilruntime.Must(mlservicev1.AddToScheme(s))
	utilruntime.Must(mltpv1.AddToScheme(s))
	return s
}

// newSuite builds the K8s client from the ambient kubeconfig and starts the
// port-forwards. It does NOT create the shared tenant — that happens in
// TestMain after a readiness gate so failures are reported clearly.
func newSuite() (*suite, error) {
	cfg := loadConfig()
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

	doer := &http.Client{Timeout: 30 * time.Second}
	user := setUser(cfg.User)
	if s.clusterManager, err = clustermanager.NewClientWithResponses(cm.localURL(),
		clustermanager.WithHTTPClient(doer), clustermanager.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build cluster-manager client: %w", err)
	}
	if s.computeService, err = computeservice.NewClientWithResponses(cs.localURL(),
		computeservice.WithHTTPClient(doer), computeservice.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build compute-service client: %w", err)
	}
	if s.artifactHub, err = artifacthub.NewClientWithResponses(ah.localURL(),
		artifacthub.WithHTTPClient(doer), artifacthub.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build artifact-hub client: %w", err)
	}
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

	// output captures kubectl's stdout+stderr so a failed forward can report
	// what kubectl actually said (RBAC error, no such service, ...) instead of a
	// bare timeout. Guarded because the scanner goroutine writes while the
	// startup path reads it on failure.
	mu     sync.Mutex
	output strings.Builder
}

func (pf *portForward) record(line string) {
	pf.mu.Lock()
	pf.output.WriteString(line)
	pf.output.WriteByte('\n')
	pf.mu.Unlock()
}

// diagnostics returns the captured kubectl output for error messages.
func (pf *portForward) diagnostics() string {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	if s := strings.TrimSpace(pf.output.String()); s != "" {
		return s
	}
	return "(no kubectl output captured)"
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
			line := sc.Text()
			pf.record(line)
			if m := forwardingRe.FindStringSubmatch(line); m != nil {
				var p int
				_, _ = fmt.Sscanf(m[1], "%d", &p)
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
			return nil, fmt.Errorf("%w; kubectl said: %s", err, pf.diagnostics())
		}
		return pf, nil
	case <-time.After(30 * time.Second):
		pf.Stop()
		return nil, fmt.Errorf("timed out waiting for kubectl port-forward to svc/%s in %s; kubectl said: %s", svc, ns, pf.diagnostics())
	}
}

func (pf *portForward) localURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", pf.localPort)
}

func (pf *portForward) Stop() {
	if pf.cmd == nil || pf.cmd.Process == nil {
		return
	}
	// Ask kubectl to shut the forward down cleanly first; SIGKILL only if it
	// doesn't exit promptly. A bare Kill() can leave the API server holding the
	// SPDY stream open briefly, which occasionally wedges a fast re-run.
	done := make(chan struct{})
	go func() { _ = pf.cmd.Wait(); close(done) }()
	_ = pf.cmd.Process.Signal(os.Interrupt)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = pf.cmd.Process.Kill()
		<-done
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
// Unstructured accessor for the external gateway-api HTTPRoute CRD
// ---------------------------------------------------------------------------

func httpRouteObj() *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	return o
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
