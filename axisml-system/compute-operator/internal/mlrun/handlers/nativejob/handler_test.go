package nativejob

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
	axislabels "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/labels"
)

func newMLRun(roleReplicas int32, modify func(*axisv1alpha1.MLRun)) *axisv1alpha1.MLRun {
	mlj := &axisv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smoke",
			Namespace: "tnt",
			Labels: map[string]string{
				axislabels.RunIDLabel: "11111111-2222-3333-4444-555555555555",
				axislabels.QuotaLabel: "training",
			},
		},
		Spec: axisv1alpha1.MLRunSpec{
			Backend: axisv1alpha1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: axisv1alpha1.SchedulingSpec{
				Quota: "axisml-tnt-default-training",
			},
			Roles: []axisv1alpha1.RoleSpec{{
				Name:          axisv1alpha1.DefaultRoleName,
				Replicas:      roleReplicas,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: axisv1alpha1.PodTemplateSubset{
					Image:   "busybox:1.36",
					Command: []string{"sh", "-c", "echo hi"},
				},
			}},
		},
	}
	if modify != nil {
		modify(mlj)
	}
	return mlj
}

func TestValidate_RequiresWorkerRole(t *testing.T) {
	h := New()
	mlj := newMLRun(1, func(m *axisv1alpha1.MLRun) {
		m.Spec.Roles[0].Name = "ps"
	})
	if errs := h.Validate(mlj); len(errs) == 0 {
		t.Fatalf("expected validation error for role name != worker")
	}
}

func TestValidate_RejectsMultiRole(t *testing.T) {
	h := New()
	mlj := newMLRun(1, func(m *axisv1alpha1.MLRun) {
		m.Spec.Roles = append(m.Spec.Roles, axisv1alpha1.RoleSpec{Name: "ps", Replicas: 1})
	})
	if errs := h.Validate(mlj); len(errs) == 0 {
		t.Fatalf("expected validation error for multi-role")
	}
}

func TestValidate_RequiresLabels(t *testing.T) {
	h := New()
	mlj := newMLRun(1, func(m *axisv1alpha1.MLRun) {
		delete(m.Labels, axislabels.RunIDLabel)
	})
	if errs := h.Validate(mlj); len(errs) == 0 {
		t.Fatalf("expected validation error for missing run-id label")
	}
}

func TestBuildJob_PodTemplateLabels(t *testing.T) {
	h := New()
	mlj := newMLRun(2, nil)
	job, err := h.buildJob(mlj)
	if err != nil {
		t.Fatalf("buildJob: %v", err)
	}
	for _, want := range []string{
		axislabels.RunIDLabel, axislabels.QuotaLabel,
		axislabels.RoleLabel, axislabels.SchedulerQuotaLabel,
	} {
		if _, ok := job.Spec.Template.Labels[want]; !ok {
			t.Errorf("Pod template missing required label %q", want)
		}
	}
	if job.Spec.Template.Spec.SchedulerName != axislabels.SchedulerName {
		t.Errorf("Pod schedulerName: want %q, got %q",
			axislabels.SchedulerName, job.Spec.Template.Spec.SchedulerName)
	}
	if job.Spec.Parallelism == nil || *job.Spec.Parallelism != 2 {
		t.Errorf("Parallelism: want 2, got %v", job.Spec.Parallelism)
	}
	if job.Spec.Completions == nil || *job.Spec.Completions != 2 {
		t.Errorf("Completions: want 2, got %v", job.Spec.Completions)
	}
}

func TestMapStatus_TransitionTable(t *testing.T) {
	h := New()
	now := metav1.Now()
	completions := int32(1)

	cases := []struct {
		name string
		job  *batchv1.Job
		want axisv1alpha1.MLRunPhase
	}{
		{
			name: "fresh job, no pods yet",
			job: &batchv1.Job{
				Spec:   batchv1.JobSpec{Parallelism: &completions},
				Status: batchv1.JobStatus{},
			},
			want: axisv1alpha1.PhasePending,
		},
		{
			name: "active pod",
			job: &batchv1.Job{
				Spec:   batchv1.JobSpec{Parallelism: &completions},
				Status: batchv1.JobStatus{Active: 1, StartTime: &now},
			},
			want: axisv1alpha1.PhaseRunning,
		},
		{
			name: "complete condition",
			job: &batchv1.Job{
				Spec: batchv1.JobSpec{Parallelism: &completions},
				Status: batchv1.JobStatus{
					Succeeded: 1,
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionTrue, LastTransitionTime: now},
					},
					CompletionTime: &now,
				},
			},
			want: axisv1alpha1.PhaseSucceeded,
		},
		{
			name: "failed condition",
			job: &batchv1.Job{
				Spec: batchv1.JobSpec{Parallelism: &completions},
				Status: batchv1.JobStatus{
					Failed: 1,
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: now, Message: "OOM"},
					},
				},
			},
			want: axisv1alpha1.PhaseFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.MapStatus(tc.job)
			if got.Phase != tc.want {
				t.Fatalf("phase: want %q, got %q (msg=%q)", tc.want, got.Phase, got.Message)
			}
		})
	}
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := batchv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := axisv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReconcile_CreatesJob(t *testing.T) {
	// First Reconcile against an empty cluster must create a Job whose
	// name matches the MLRun name and whose Pod template carries the
	// mandatory labels + axisml-scheduler.
	s := newScheme(t)
	c := fakeclient.NewClientBuilder().WithScheme(s).Build()
	h := New()
	mlj := newMLRun(2, nil)

	if _, _, err := h.Reconcile(context.Background(), c, mlj); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var got batchv1.Job
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "smoke"}, &got); err != nil {
		t.Fatalf("Job not created: %v", err)
	}
	if got.Spec.Parallelism == nil || *got.Spec.Parallelism != 2 {
		t.Fatalf("Parallelism: want 2, got %v", got.Spec.Parallelism)
	}
	if got.Spec.Template.Spec.SchedulerName != axislabels.SchedulerName {
		t.Fatalf("scheduler: want %q, got %q",
			axislabels.SchedulerName, got.Spec.Template.Spec.SchedulerName)
	}
	for _, want := range []string{
		axislabels.RunIDLabel, axislabels.QuotaLabel,
		axislabels.RoleLabel, axislabels.SchedulerQuotaLabel,
	} {
		if _, ok := got.Spec.Template.Labels[want]; !ok {
			t.Errorf("Pod template missing required label %q", want)
		}
	}
}

func TestReconcile_CancelBeforeCreate(t *testing.T) {
	// Suspend=true on a CR with no underlying Job must NOT create the
	// Job; the handler must report SuspendCompleted so the dispatcher
	// writes the Suspended condition immediately.
	s := newScheme(t)
	c := fakeclient.NewClientBuilder().WithScheme(s).Build()
	h := New()
	mlj := newMLRun(1, func(m *axisv1alpha1.MLRun) {
		m.Spec.RunPolicy.Suspend = true
	})

	underlying, recRes, err := h.Reconcile(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if underlying != nil {
		t.Errorf("underlying must be nil on cancel-before-create, got %T", underlying)
	}
	if !recRes.SuspendCompleted {
		t.Errorf("SuspendCompleted must be set on cancel-before-create")
	}
	if recRes.SuspendReason != axisv1alpha1.ReasonCancelRequested {
		t.Errorf("SuspendReason: want CancelRequested, got %q", recRes.SuspendReason)
	}

	var jobs batchv1.JobList
	if err := c.List(context.Background(), &jobs); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs.Items) != 0 {
		t.Fatalf("Job must NOT be created on cancel-before-create, got %d", len(jobs.Items))
	}
}

func TestReconcile_SuspendsExistingJob(t *testing.T) {
	// Suspend=true with an existing Job must patch Job.spec.suspend=true
	// (Kubernetes evicts running Pods) and report SuspendCompleted.
	s := newScheme(t)
	mlj := newMLRun(1, nil)
	h := New()

	// Seed a Job that mirrors what an earlier Reconcile would have
	// produced. We rebuild via h.buildJob to keep the test honest.
	job, err := h.buildJob(mlj)
	if err != nil {
		t.Fatalf("buildJob: %v", err)
	}
	c := fakeclient.NewClientBuilder().WithScheme(s).WithObjects(job).Build()

	mlj.Spec.RunPolicy.Suspend = true
	got, recRes, err := h.Reconcile(context.Background(), c, mlj)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !recRes.SuspendCompleted {
		t.Fatalf("SuspendCompleted must be set after suspending existing Job")
	}
	gotJob, ok := got.(*batchv1.Job)
	if !ok {
		t.Fatalf("underlying type: %T", got)
	}
	if gotJob.Spec.Suspend == nil || !*gotJob.Spec.Suspend {
		t.Fatalf("Job.spec.suspend must be true after suspend reconcile, got %v", gotJob.Spec.Suspend)
	}
}

func TestReconcile_IsIdempotent(t *testing.T) {
	// Repeated Reconcile with unchanged spec must not recreate the
	// Job (no apierrors.IsAlreadyExists in the wild, no resource churn).
	s := newScheme(t)
	c := fakeclient.NewClientBuilder().WithScheme(s).Build()
	h := New()
	mlj := newMLRun(2, nil)

	if _, _, err := h.Reconcile(context.Background(), c, mlj); err != nil {
		t.Fatalf("first Reconcile: %v", err)
	}
	var first batchv1.Job
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "smoke"}, &first); err != nil {
		t.Fatalf("first get: %v", err)
	}
	firstUID := first.UID

	if _, _, err := h.Reconcile(context.Background(), c, mlj); err != nil {
		t.Fatalf("second Reconcile: %v", err)
	}
	var second batchv1.Job
	if err := c.Get(context.Background(), client.ObjectKey{Namespace: "tnt", Name: "smoke"}, &second); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if second.UID != firstUID {
		t.Fatalf("Job recreated across reconciles: UID %q → %q", firstUID, second.UID)
	}
}

// Compile-time assertion that the runtime.Object interface tree is wired
// correctly (deepcopy not yet generated).
var _ runtime.Object = &axisv1alpha1.MLRun{}
