//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlrunv1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// compute-operator. MLRun/MLService CRs are applied DIRECTLY (bypassing
// compute-service) to isolate the operator, into the shared tenant namespace.

func TestComputeOperator_MLRunRunsToCompletion(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-job")

	job := buildMLRunCR(ns, name, quota)
	require.NoError(t, h.k8s.Create(ctx, job))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), job) })

	// MLRun reaches Succeeded once kubelet has actually run the pod.
	eventually(t, h.cfg.MLRunCompleteTimeout, func() error {
		var cur mlrunv1.MLRun
		if err := h.get(ctx, ns, name, &cur); err != nil {
			return err
		}
		if cur.Status.Phase != mlrunv1.PhaseSucceeded {
			return assertErr("phase=%q want Succeeded", cur.Status.Phase)
		}
		return nil
	})
}

func TestComputeOperator_SchedulerAndQuotaLabels(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-sched")

	job := buildMLRunCR(ns, name, quota)
	require.NoError(t, h.k8s.Create(ctx, job))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), job) })

	// Once a pod exists, it must carry koord-scheduler + the koord quota label
	// (the non-negotiable invariant from CLAUDE.md).
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		pod, err := firstPodMatching(ctx, ns, name)
		if err != nil {
			return err
		}
		if pod == nil {
			return assertErr("no pod for job %s yet", name)
		}
		if pod.Spec.SchedulerName != "koord-scheduler" {
			return assertErr("schedulerName=%q want koord-scheduler", pod.Spec.SchedulerName)
		}
		if _, ok := pod.Labels["quota.scheduling.koordinator.sh/name"]; !ok {
			return assertErr("pod missing koord quota label")
		}
		return nil
	})
}

func TestComputeOperator_MLRunCancelSuspends(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-cancel")

	// Long-running so we can observe suspension.
	job := buildMLRunCR(ns, name, quota)
	job.Spec.Roles[0].Template.Command = []string{"sh", "-c", "sleep 600"}
	require.NoError(t, h.k8s.Create(ctx, job))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), job) })

	// Cancel by setting spec.runPolicy.suspend via an unstructured patch (avoids
	// depending on the exact Go field tag).
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return setMLRunSuspend(ctx, ns, name, true) })

	// The backing batch/v1 Job is suspended.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		suspended, err := batchJobSuspended(ctx, ns, name)
		if err != nil {
			return err
		}
		if !suspended {
			return assertErr("batch Job %s not suspended", name)
		}
		return nil
	})
}

func TestComputeOperator_MLServiceDeploymentServes(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-svc")

	svc := buildMLServiceCR(ns, name, quota, "deployment", nil)
	require.NoError(t, h.k8s.Create(ctx, svc))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), svc) })

	// Deployment + Service exist and the MLService becomes Ready.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var dep appsv1.Deployment
		if err := h.get(ctx, ns, name, &dep); err != nil {
			return err
		}
		var k8ssvc corev1.Service
		if err := h.get(ctx, ns, name, &k8ssvc); err != nil {
			return err
		}
		var cur mlservicev1.MLService
		if err := h.get(ctx, ns, name, &cur); err != nil {
			return err
		}
		if cur.Status.Phase != mlservicev1.PhaseReady {
			return assertErr("phase=%q want Ready", cur.Status.Phase)
		}
		return nil
	})

	// Real HTTP: port-forward to the in-namespace Service and GET / -> 200.
	pf, err := startPortForward(ns, name, 80)
	require.NoError(t, err)
	defer pf.Stop()
	cli := newHTTPClient(pf.localURL(), "")
	r, err := cli.do(ctx, http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.status, "nginx should answer 200")
}

func TestComputeOperator_MLServiceRouteThroughEnvoy(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-route")

	route := &mlservicev1.Route{Enabled: true, Hostname: name + ".e2e.local"}
	svc := buildMLServiceCR(ns, name, quota, "deployment", route)
	require.NoError(t, h.k8s.Create(ctx, svc))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), svc) })

	// An HTTPRoute is created with parentRef -> axisml-gateway/axisml-infra.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		hr := httpRouteObj()
		if err := h.get(ctx, ns, name, hr); err != nil {
			return err
		}
		parents, found, _ := unstructured.NestedSlice(hr.Object, "spec", "parentRefs")
		if !found || len(parents) == 0 {
			return assertErr("HTTPRoute %s has no parentRefs", name)
		}
		m, _ := parents[0].(map[string]any)
		if m["name"] != "axisml-gateway" {
			return assertErr("parentRef name=%v want axisml-gateway", m["name"])
		}
		return nil
	})
	// NOTE: driving an actual request through the Envoy LB requires resolving
	// the minikube gateway address; left to a manual/follow-up check.
	t.Log("HTTPRoute wired to axisml-gateway; end-to-end gateway curl not asserted")
}

func TestComputeOperator_StatefulSetEngine(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-sts")

	svc := buildMLServiceCR(ns, name, quota, "statefulset", nil)
	require.NoError(t, h.k8s.Create(ctx, svc))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), svc) })

	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var sts appsv1.StatefulSet
		if err := h.get(ctx, ns, name, &sts); err != nil {
			return err
		}
		var cur mlservicev1.MLService
		if err := h.get(ctx, ns, name, &cur); err != nil {
			return err
		}
		if cur.Status.Phase != mlservicev1.PhaseReady {
			return assertErr("phase=%q want Ready", cur.Status.Phase)
		}
		return nil
	})
}

func TestComputeOperator_MLServiceScaleViaCR(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)
	name := uniqueName("e2e-scale")

	svc := buildMLServiceCR(ns, name, quota, "deployment", nil)
	require.NoError(t, h.k8s.Create(ctx, svc))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), svc) })

	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var dep appsv1.Deployment
		return h.get(ctx, ns, name, &dep)
	})

	// Scale 1 -> 2 by updating the CR.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var cur mlservicev1.MLService
		if err := h.get(ctx, ns, name, &cur); err != nil {
			return err
		}
		cur.Spec.Roles[0].Replicas = 2
		return h.k8s.Update(ctx, &cur)
	})
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var dep appsv1.Deployment
		if err := h.get(ctx, ns, name, &dep); err != nil {
			return err
		}
		if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 2 {
			return assertErr("deployment replicas not 2")
		}
		return nil
	})
}

func TestComputeOperator_OverQuotaMLRunPends(t *testing.T) {
	ctx := context.Background()
	tn := uniqueName("e2e-oq-tenant")
	ns := tn
	// 1 CPU quota.
	createTenantCR(t, ctx, buildTenant(tn, ns, h.cfg.DefaultPool, "1", "2Gi"))
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.namespaceExists(ctx, ns) })
	var quota string
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		names, err := elasticQuotaNames(ctx, ns)
		if err != nil || len(names) == 0 {
			return assertErr("no quota yet (err=%v)", err)
		}
		quota = names[0]
		return nil
	})

	name := uniqueName("e2e-oq")
	job := buildMLRunCR(ns, name, quota)
	// Request more CPU than the quota allows.
	job.Spec.Roles[0].Template.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: mustQty("4")},
	}
	require.NoError(t, h.k8s.Create(ctx, job))
	t.Cleanup(func() { _ = h.k8s.Delete(context.Background(), job) })

	// The pod must stay Pending (unscheduled) on the ElasticQuota.
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		pod, err := firstPodMatching(ctx, ns, name)
		if err != nil {
			return err
		}
		if pod == nil {
			return assertErr("no pod yet")
		}
		if pod.Spec.NodeName != "" {
			return assertErr("pod unexpectedly scheduled to %s", pod.Spec.NodeName)
		}
		return nil
	})
}

// ---- compute-operator helpers ----

func buildMLRunCR(ns, name, quota string) *mlrunv1.MLRun {
	return &mlrunv1.MLRun{
		// The operator validates that compute-service's identity labels are
		// present (run-id is mandatory); tenant/quota mirror the namespace +
		// ElasticQuota. We use the CR name as the id.
		ObjectMeta: objMeta(ns, name, map[string]string{
			mlrunv1.LabelRunID:  name,
			mlrunv1.LabelTenant: ns,
			mlrunv1.LabelQuota:  quota,
		}),
		Spec: mlrunv1.MLRunSpec{
			Backend:    mlrunv1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: mlrunv1.SchedulingSpec{Quota: quota},
			Roles: []mlrunv1.RoleSpec{{
				Name:          mlrunv1.DefaultRoleName,
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: mlrunv1.PodTemplateSubset{
					Image:           h.cfg.MLRunImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
}

func buildMLServiceCR(ns, name, quota, engine string, route *mlservicev1.Route) *mlservicev1.MLService {
	return &mlservicev1.MLService{
		ObjectMeta: objMeta(ns, name, map[string]string{
			mlservicev1.LabelServiceID: name,
			mlservicev1.LabelTenant:    ns,
			mlservicev1.LabelQuota:     quota,
		}),
		Spec: mlservicev1.MLServiceSpec{
			Backend:    mlservicev1.Backend{Name: "native", Engine: engine},
			Scheduling: mlservicev1.Scheduling{Quota: quota},
			Roles: []mlservicev1.RoleSpec{{
				Name:     mlservicev1.DefaultRoleName,
				Replicas: 1,
				Template: mlservicev1.PodTemplate{
					Image:           h.cfg.ServiceImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports:           []mlservicev1.PodPort{{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP}},
				},
			}},
			Route: route,
		},
	}
}

// firstPodMatching returns the first pod in ns whose name contains sub.
func firstPodMatching(ctx context.Context, ns, sub string) (*corev1.Pod, error) {
	var pods corev1.PodList
	if err := h.k8s.List(ctx, &pods, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	for i := range pods.Items {
		if strings.Contains(pods.Items[i].Name, sub) {
			return &pods.Items[i], nil
		}
	}
	return nil, nil
}

// setMLRunSuspend patches spec.runPolicy.suspend on the MLRun via unstructured.
func setMLRunSuspend(ctx context.Context, ns, name string, suspend bool) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(mlrunv1.GroupVersion.WithKind("MLRun"))
	if err := h.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, obj); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(obj.Object, suspend, "spec", "runPolicy", "suspend"); err != nil {
		return err
	}
	return h.k8s.Update(ctx, obj)
}

// batchJobSuspended reports whether the backing batch/v1 Job has spec.suspend.
func batchJobSuspended(ctx context.Context, ns, name string) (bool, error) {
	job := newBatchJob()
	if err := h.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, job); err != nil {
		return false, err
	}
	suspend, found, _ := unstructured.NestedBool(job.Object, "spec", "suspend")
	return found && suspend, nil
}
