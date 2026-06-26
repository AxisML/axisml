//go:build e2e || standard || lite

package e2e

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// Form-neutral request builders + teardown helpers shared by the CORE black-box
// tests. They use only the generated client DTOs and string literals (never the
// compute-operator Go API constants), so this file carries no Kubernetes import
// and compiles under the lite build. The literal values mirror the operator API
// (role names "worker"/"predictor", restart "Never", kind "service", ...).

var nameSeq int64

// ptr returns a pointer to v. The generated clients model optional fields as
// pointers, so request builders lean on this heavily.
func ptr[T any](v T) *T { return &v }

// uniqueName returns a short, DNS-safe, run-unique name with the given prefix.
func uniqueName(prefix string) string {
	n := atomic.AddInt64(&nameSeq, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix()%100000, n)
}

// busyboxMLRunReq builds a minimal native/job MLRun create request that runs to
// completion quickly (echoes a marker the log test asserts on).
func busyboxMLRunReq(name, pool, unit, quota string) computeservice.MLRunCreateRequest {
	return computeservice.MLRunCreateRequest{
		Name:     name,
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &computeservice.MLRunBackendSpec{Name: "native", Engine: "job"},
		Roles: []computeservice.MLRunRoleSpec{{
			Name:          "worker",
			Replicas:      1,
			RestartPolicy: ptr("Never"),
			Template: computeservice.MLRunPodTemplateSubset{
				Image:           h.config().MLRunImage,
				ImagePullPolicy: ptr("IfNotPresent"),
				Command:         &[]string{"sh", "-c", "echo hello"},
			},
		}},
	}
}

// nginxMLServiceReq builds a minimal native/deployment MLService create request.
func nginxMLServiceReq(name, pool, unit, quota string, route *computeservice.MLServiceRoute) computeservice.MLServiceCreateRequest {
	return computeservice.MLServiceCreateRequest{
		Name:     name,
		Kind:     ptr("service"),
		PoolName: pool,
		UnitName: unit,
		Quota:    quota,
		Backend:  &computeservice.MLServiceBackend{Name: "native", Engine: "deployment"},
		Roles: []computeservice.MLServiceRoleSpec{{
			Name:     "predictor",
			Replicas: 1,
			Template: computeservice.MLServicePodTemplate{
				Image:           h.config().ServiceImage,
				ImagePullPolicy: ptr("IfNotPresent"),
				Ports:           &[]computeservice.MLServicePodPort{{Name: "http", ContainerPort: 80, Protocol: ptr("TCP")}},
			},
		}},
		Route: route,
	}
}

// canaryTrafficReq builds a canary traffic policy fronting two member services
// (stable @90 / canary @10).
func canaryTrafficReq(name, stableSvc, canarySvc string) computeservice.TrafficPolicyCreateRequest {
	return computeservice.TrafficPolicyCreateRequest{
		Name:     name,
		Mode:     "canary",
		Endpoint: &computeservice.MLTrafficPolicyEndpoint{Hostname: ptr(name + ".e2e.local")},
		Backends: []computeservice.MLTrafficPolicyBackendMember{
			{ServiceName: stableSvc, Role: ptr("stable"), Weight: 90},
			{ServiceName: canarySvc, Role: ptr("canary"), Weight: 10},
		},
	}
}

// cleanupMLRun registers best-effort black-box teardown: the user-facing DELETE.
// Unlike the old Standard helper there is no Kubernetes fallback — the CORE tests
// only know the HTTP contract, and a leaked CR is the form's own concern.
func cleanupMLRun(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = h.ComputeService().DeleteMLRunWithResponse(context.Background(), ns, name)
	})
}

// cleanupMLService is the MLService analogue of cleanupMLRun.
func cleanupMLService(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = h.ComputeService().DeleteMLServiceWithResponse(context.Background(), ns, name)
	})
}

// cleanupTrafficPolicy is the MLTrafficPolicy analogue of cleanupMLRun.
func cleanupTrafficPolicy(t *testing.T, ns, name string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = h.ComputeService().DeleteTrafficPolicyWithResponse(context.Background(), ns, name)
	})
}
