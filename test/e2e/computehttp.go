package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// ComputeHTTP exposes the deployed compute service to e2e tests via
// `kubectl port-forward`. The forwarder runs as a child process for the
// life of the returned ComputeHTTP — Stop terminates it. Tests use
// BaseURL() to build request URLs and Do() / DoJSON() for one-shot calls.
//
// We deliberately shell out to kubectl rather than using client-go's
// portforward package because the SPDY upgrade dance is brittle on
// minikube ("connection refused after startup probe") and kubectl already
// re-tries internally.
type ComputeHTTP struct {
	cmd     *exec.Cmd
	port    int
	stop    func()
	stdout  *bytes.Buffer
	mu      sync.Mutex
	stopped bool
}

// PortForwardCompute starts a `kubectl port-forward` against the deployed
// compute Service (<release>-compute on port 8081 in the axisml-system
// namespace) and waits until the local port is reachable. The Service
// name follows the helm-installed convention; if the rendered name
// changes, update HelmRelease() / SystemNamespace and the svc string.
//
// Caller MUST defer Stop() to clean up the kubectl subprocess.
func PortForwardCompute(t *testing.T) *ComputeHTTP {
	t.Helper()

	svc := fmt.Sprintf("svc/%s-compute", HelmRelease())
	// `:8081` asks kubectl to bind a random local port and forward to 8081
	// in the Pod. We parse the chosen port from stdout below.
	cmd := exec.Command("kubectl", "port-forward",
		"-n", SystemNamespace, svc, ":8081")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("kubectl stdout pipe: %v", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		t.Fatalf("kubectl port-forward start: %v", err)
	}

	out := &bytes.Buffer{}
	port := waitForForwardingLine(t, stdout, out, 30*time.Second)

	c := &ComputeHTTP{cmd: cmd, port: port, stdout: out}
	c.stop = func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.stopped {
			return
		}
		c.stopped = true
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
	t.Cleanup(c.stop)

	// Probe /healthz so callers don't race kubectl's startup.
	if err := c.waitHealthy(15 * time.Second); err != nil {
		c.stop()
		t.Fatalf("compute /healthz never returned 200: %v\nport-forward log:\n%s", err, out.String())
	}
	return c
}

// Stop terminates the kubectl port-forward process. Idempotent.
func (c *ComputeHTTP) Stop() { c.stop() }

// BaseURL returns http://127.0.0.1:<port>.
func (c *ComputeHTTP) BaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.port)
}

// DoJSON is a tiny convenience: POST/PATCH/PUT a JSON body, decode the
// response into out (when non-nil), return the http.Response. Tests use
// this to keep request boilerplate out of the test bodies.
func (c *ComputeHTTP) DoJSON(t *testing.T, ctx context.Context, method, path string, body any, out any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode body: %v", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL()+path, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if out != nil {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		// Re-create the body so callers that inspect resp.StatusCode + raw
		// have something to log on failure.
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, out); err != nil {
				t.Fatalf("decode %s %s (status=%d body=%s): %v", method, path, resp.StatusCode, string(raw), err)
			}
		}
	}
	return resp
}

// ReadBody is a defensive helper: callers occasionally need both decode
// and the raw bytes for failure messages. ReadBody drains and closes
// resp.Body and returns the bytes; callers must not read the body again
// after this call.
func ReadBody(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return raw
}

func (c *ComputeHTTP) waitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, c.BaseURL()+"/healthz", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for compute /healthz")
}

// waitForForwardingLine consumes kubectl's stdout until the
// "Forwarding from 127.0.0.1:<port> -> 8081" announcement appears, then
// keeps draining stdout in the background so kubectl's pipe doesn't fill
// and stall the forwarder. Returns the bound local port.
func waitForForwardingLine(t *testing.T, stdout io.Reader, sink *bytes.Buffer, timeout time.Duration) int {
	t.Helper()
	pattern := regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)\s*->`)

	type result struct {
		port int
		err  error
	}
	ch := make(chan result, 1)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1<<14), 1<<20)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			sink.WriteString(line + "\n")
			if m := pattern.FindStringSubmatch(line); m != nil {
				var p int
				_, _ = fmt.Sscanf(m[1], "%d", &p)
				ch <- result{port: p}
				// keep draining so kubectl doesn't block on the pipe.
				for scanner.Scan() {
					sink.WriteString(scanner.Text() + "\n")
				}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- result{err: err}
		} else {
			ch <- result{err: fmt.Errorf("kubectl exited before announcing port; log: %s", sink.String())}
		}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("port-forward: %v", r.err)
		}
		return r.port
	case <-time.After(timeout):
		t.Fatalf("port-forward never announced port within %s\nlog:\n%s",
			timeout, sink.String())
		return 0
	}
}

// PrettyResp returns a one-line status + body summary for failure logs.
func PrettyResp(resp *http.Response, body []byte) string {
	if resp == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d %s body=%s",
		resp.StatusCode, http.StatusText(resp.StatusCode),
		strings.TrimSpace(string(body)))
}
