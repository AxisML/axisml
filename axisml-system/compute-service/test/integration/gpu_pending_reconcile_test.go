//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/mlrun"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/mlservice"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
	"github.com/axisml/axisml/test/testutil"
)

// fakeResourceRuntime is a controllable extensions.ComputeRuntime standing in for
// a standalone runtime. Apply* returns ResourceUnavailableError while
// `unavailable` is set (no free GPU), and Observe* reports "no containers"
// (NotFound) until `placed` is flipped, at which point the workload is Running/
// Ready. Only the Run/Service Apply+Observe+Cancel+Delete methods are exercised
// by the reconciler/poller under test; the rest panic if ever called.
type fakeResourceRuntime struct {
	mu          sync.Mutex
	unavailable bool
	placed      bool
	applyError  error
}

func (f *fakeResourceRuntime) setPlaced() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unavailable = false
	f.placed = true
}

func (f *fakeResourceRuntime) applyErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.applyError != nil {
		return f.applyError
	}
	if f.unavailable {
		return extensions.NewResourceUnavailable("等待可用 GPU（需 1，空闲 0）")
	}
	return nil
}

func (f *fakeResourceRuntime) isPlaced() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.placed
}

func (f *fakeResourceRuntime) ApplyMLRun(context.Context, *mlrunv1alpha1.MLRun) error {
	return f.applyErr()
}

func (f *fakeResourceRuntime) ObserveMLRun(_ context.Context, key types.NamespacedName) (mlrunv1alpha1.MLRunStatus, error) {
	if f.isPlaced() {
		return mlrunv1alpha1.MLRunStatus{
			Phase:     mlrunv1alpha1.PhaseRunning,
			StartedAt: &metav1.Time{Time: time.Now().UTC()},
		}, nil
	}
	return mlrunv1alpha1.MLRunStatus{}, apierrors.NewNotFound(schema.GroupResource{Group: "axisml.io", Resource: "mlruns"}, key.Name)
}

func (f *fakeResourceRuntime) ApplyMLService(context.Context, *mlservicev1alpha1.MLService) error {
	return f.applyErr()
}

func (f *fakeResourceRuntime) ObserveMLService(_ context.Context, key types.NamespacedName) (mlservicev1alpha1.MLServiceStatus, error) {
	if f.isPlaced() {
		return mlservicev1alpha1.MLServiceStatus{Phase: mlservicev1alpha1.PhaseReady, ReadyReplicas: 1}, nil
	}
	return mlservicev1alpha1.MLServiceStatus{}, apierrors.NewNotFound(schema.GroupResource{Group: "axisml.io", Resource: "mlservices"}, key.Name)
}

// --- unused interface methods (never called by the reconciler/poller path) ---

func (f *fakeResourceRuntime) CancelMLRun(context.Context, types.NamespacedName) error { return nil }
func (f *fakeResourceRuntime) DeleteMLRun(context.Context, types.NamespacedName) error { return nil }
func (f *fakeResourceRuntime) DeleteMLService(context.Context, types.NamespacedName) error {
	return nil
}

func (f *fakeResourceRuntime) ListMLRunInstances(context.Context, types.NamespacedName) (*corev1.PodList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLRunInstanceLogs(context.Context, types.NamespacedName, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLRunInstanceEvents(context.Context, types.NamespacedName, string) (*eventsv1.EventList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLRunEvents(context.Context, types.NamespacedName) (*eventsv1.EventList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) ListMLServiceInstances(context.Context, types.NamespacedName) (*corev1.PodList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLServiceInstanceLogs(context.Context, types.NamespacedName, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLServiceInstanceEvents(context.Context, types.NamespacedName, string) (*eventsv1.EventList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLServiceEvents(context.Context, types.NamespacedName) (*eventsv1.EventList, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) ApplyMLTrafficPolicy(context.Context, *mltrafficpolicyv1alpha1.MLTrafficPolicy) error {
	panic("unused")
}
func (f *fakeResourceRuntime) ObserveMLTrafficPolicy(context.Context, types.NamespacedName) (mltrafficpolicyv1alpha1.MLTrafficPolicyStatus, error) {
	panic("unused")
}
func (f *fakeResourceRuntime) DeleteMLTrafficPolicy(context.Context, types.NamespacedName) error {
	panic("unused")
}
func (f *fakeResourceRuntime) GetMLTrafficPolicyEvents(context.Context, types.NamespacedName) (*eventsv1.EventList, error) {
	panic("unused")
}

var _ extensions.ComputeRuntime = (*fakeResourceRuntime)(nil)

func runPhase(t *testing.T, ctx context.Context, id uuid.UUID) string {
	t.Helper()
	var row store.MLRun
	require.NoError(t, gormDB.WithContext(ctx).First(&row, "id = ?", id).Error)
	return row.Phase
}

func servicePhase(t *testing.T, ctx context.Context, id uuid.UUID) string {
	t.Helper()
	var row store.MLService
	require.NoError(t, gormDB.WithContext(ctx).First(&row, "id = ?", id).Error)
	return row.Phase
}

// TestMLRun_WaitsForGPUThenRuns drives the "no free GPU → Pending → retry →
// Running" state machine through the real mlrun Reconciler + StatusPoller with a
// fake runtime: the Run must reach Pending, STAY Pending (the poller must not
// cancel a container-less Pending Run), then advance to Running once a card frees.
func TestMLRun_WaitsForGPUThenRuns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := &fakeResourceRuntime{unavailable: true}
	recon := mlrun.NewReconciler(gormDB, fake, logr.Discard(), 50*time.Millisecond, false)
	poller := mlrun.NewStatusPoller(gormDB, fake, logr.Discard(), 50*time.Millisecond)
	go func() { _ = recon.Start(ctx) }()
	go func() { _ = poller.Start(ctx) }()

	id := uuid.New()
	require.NoError(t, gormDB.WithContext(ctx).Create(&store.MLRun{
		ID:          id,
		Namespace:   "gpu-pending-it-run", // not created in envtest; isolates from the global pipeline
		Name:        "waiter-" + id.String()[:8],
		Spec:        datatypes.JSON(`{}`),
		Phase:       string(mlrun.StatusCreating),
		Labels:      datatypes.JSON(`{}`),
		Annotations: datatypes.JSON(`{}`),
		StatusJSON:  datatypes.JSON(`{}`),
	}).Error)

	// Reaches Pending (pickup + no free GPU).
	testutil.Eventually(t, 5*time.Second, 50*time.Millisecond, func() error {
		if p := runPhase(t, ctx, id); p != string(mlrun.StatusPending) {
			return fmt.Errorf("phase=%s, want Pending", p)
		}
		return nil
	})

	// STAYS Pending: the poller Observes NotFound (no containers) but must not
	// converge a waiting Run to Cancelled.
	deadline := time.Now().Add(600 * time.Millisecond)
	for time.Now().Before(deadline) {
		require.Equal(t, string(mlrun.StatusPending), runPhase(t, ctx, id),
			"a Run waiting for a GPU must stay Pending, never be cancelled")
		time.Sleep(60 * time.Millisecond)
	}

	// A card frees → Apply succeeds and the runtime reports Running.
	fake.setPlaced()
	testutil.Eventually(t, 5*time.Second, 50*time.Millisecond, func() error {
		if p := runPhase(t, ctx, id); p != string(mlrun.StatusRunning) {
			return fmt.Errorf("phase=%s, want Running", p)
		}
		return nil
	})
}

func TestMLRun_TerminalApplyErrorFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	applyErr := errors.New(`pull image "registry.example.com/missing:v1": manifest unknown`)
	fake := &fakeResourceRuntime{applyError: extensions.NewTerminalApplyError(applyErr)}
	recon := mlrun.NewReconciler(gormDB, fake, logr.Discard(), 50*time.Millisecond, false)
	go func() { _ = recon.Start(ctx) }()

	id := uuid.New()
	require.NoError(t, gormDB.WithContext(ctx).Create(&store.MLRun{
		ID:          id,
		Namespace:   "terminal-apply-it-run",
		Name:        "failed-" + id.String()[:8],
		Spec:        datatypes.JSON(`{}`),
		Phase:       string(mlrun.StatusCreating),
		Labels:      datatypes.JSON(`{}`),
		Annotations: datatypes.JSON(`{}`),
		StatusJSON:  datatypes.JSON(`{}`),
	}).Error)

	testutil.Eventually(t, 5*time.Second, 50*time.Millisecond, func() error {
		var row store.MLRun
		if err := gormDB.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		if row.Phase != string(mlrun.StatusFailed) {
			return fmt.Errorf("phase=%s, want Failed", row.Phase)
		}
		var status server.MLRunStatus
		if err := json.Unmarshal(row.StatusJSON, &status); err != nil {
			return err
		}
		if status.Message != applyErr.Error() {
			return fmt.Errorf("message=%q, want %q", status.Message, applyErr.Error())
		}
		if status.FinishedAt == nil {
			return errors.New("finishedAt is nil")
		}
		return nil
	})
}

// TestMLService_WaitingNotSilentlyDeleted guards the reflectGone regression: a
// GPU-waiting Service (Pending, no containers) must NOT be pushed to
// Deleting/Deleted by the poller, and must reach Ready once a card frees.
func TestMLService_WaitingNotSilentlyDeleted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fake := &fakeResourceRuntime{unavailable: true}
	recon := mlservice.NewReconciler(gormDB, fake, logr.Discard(), 50*time.Millisecond, false)
	poller := mlservice.NewStatusPoller(gormDB, fake, logr.Discard(), 50*time.Millisecond)
	go func() { _ = recon.Start(ctx) }()
	go func() { _ = poller.Start(ctx) }()

	id := uuid.New()
	require.NoError(t, gormDB.WithContext(ctx).Create(&store.MLService{
		ID:          id,
		Namespace:   "gpu-pending-it-svc",
		Name:        "waiter-" + id.String()[:8],
		Kind:        "service",
		Spec:        datatypes.JSON(`{"roles":[{"replicas":1}]}`),
		Phase:       string(mlservice.StatusCreating),
		Labels:      datatypes.JSON(`{}`),
		Annotations: datatypes.JSON(`{}`),
		StatusJSON:  datatypes.JSON(`{}`),
		Generation:  1,
	}).Error)

	testutil.Eventually(t, 5*time.Second, 50*time.Millisecond, func() error {
		if p := servicePhase(t, ctx, id); p != string(mlservice.StatusPending) {
			return fmt.Errorf("phase=%s, want Pending", p)
		}
		return nil
	})

	// STAYS Pending — never silently deleted (reflectGone must skip Pending).
	deadline := time.Now().Add(600 * time.Millisecond)
	for time.Now().Before(deadline) {
		p := servicePhase(t, ctx, id)
		require.NotContains(t, []string{string(mlservice.StatusDeleting), string(mlservice.StatusDeleted)}, p,
			"a Service waiting for a GPU must not be silently deleted")
		require.Equal(t, string(mlservice.StatusPending), p)
		time.Sleep(60 * time.Millisecond)
	}

	fake.setPlaced()
	testutil.Eventually(t, 5*time.Second, 50*time.Millisecond, func() error {
		if p := servicePhase(t, ctx, id); p != string(mlservice.StatusReady) {
			return fmt.Errorf("phase=%s, want Ready", p)
		}
		return nil
	})
}
