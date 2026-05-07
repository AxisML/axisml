package handler

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	axislabels "github.com/axisml/axisml/components/compute-operator/internal/mljob/labels"
)

func validMLJob() *axisv1alpha1.MLJob {
	return &axisv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "training-1",
			Namespace: "team-a",
			UID:       types.UID("uid-1"),
			Labels: map[string]string{
				axislabels.JobIDLabel: "job-1",
				axislabels.QuotaLabel: "default",
			},
		},
		Spec: axisv1alpha1.MLJobSpec{
			Scheduling: axisv1alpha1.SchedulingSpec{
				Quota:         "axisml-team-a-default-q1",
				PriorityClass: "high",
				NodeSelector:  map[string]string{"gpu": "true"},
				Tolerations:   []corev1.Toleration{{Key: "dedicated", Value: "training"}},
			},
		},
	}
}

func TestEnsureRequiredCRLabels_NilLabels(t *testing.T) {
	mlj := &axisv1alpha1.MLJob{}
	errs := EnsureRequiredCRLabels(mlj)
	if len(errs) == 0 {
		t.Fatal("expected error for nil labels")
	}
}

func TestEnsureRequiredCRLabels_MissingKey(t *testing.T) {
	mlj := &axisv1alpha1.MLJob{}
	mlj.Labels = map[string]string{axislabels.JobIDLabel: "job-1"} // missing quota
	errs := EnsureRequiredCRLabels(mlj)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error; got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Detail, "Compute") {
		t.Errorf("error detail should reference Compute; got %q", errs[0].Detail)
	}
}

func TestEnsureRequiredCRLabels_AllPresent(t *testing.T) {
	if errs := EnsureRequiredCRLabels(validMLJob()); len(errs) != 0 {
		t.Errorf("expected no errors; got %v", errs)
	}
}

func TestInjectAxisMLLabels_PopulatesLabelsAndScheduling(t *testing.T) {
	mlj := validMLJob()
	role := axisv1alpha1.RoleSpec{Name: "worker"}
	tmpl := &corev1.PodTemplateSpec{}

	if err := InjectAxisMLLabels(tmpl, mlj, role, map[string]string{"extra": "true"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wants := map[string]string{
		axislabels.JobIDLabel:      "job-1",
		axislabels.QuotaLabel:      "default",
		axislabels.RoleLabel:       "worker",
		axislabels.KoordQuotaLabel: "axisml-team-a-default-q1",
		"extra":                    "true",
	}
	for k, v := range wants {
		if tmpl.Labels[k] != v {
			t.Errorf("label %s = %q; want %q", k, tmpl.Labels[k], v)
		}
	}
	if tmpl.Spec.SchedulerName != axislabels.KoordSchedulerName {
		t.Errorf("scheduler = %q; want %q (Koord scheduler is mandatory for ElasticQuota)",
			tmpl.Spec.SchedulerName, axislabels.KoordSchedulerName)
	}
	if tmpl.Spec.PriorityClassName != "high" {
		t.Errorf("PriorityClass = %q", tmpl.Spec.PriorityClassName)
	}
	if tmpl.Spec.NodeSelector["gpu"] != "true" {
		t.Errorf("nodeSelector not propagated: %v", tmpl.Spec.NodeSelector)
	}
	if len(tmpl.Spec.Tolerations) != 1 || tmpl.Spec.Tolerations[0].Key != "dedicated" {
		t.Errorf("tolerations not propagated: %v", tmpl.Spec.Tolerations)
	}
}

func TestInjectAxisMLLabels_PreservesExistingLabels(t *testing.T) {
	mlj := validMLJob()
	tmpl := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"keep": "1"}},
	}
	if err := InjectAxisMLLabels(tmpl, mlj, axisv1alpha1.RoleSpec{Name: "ps"}, nil); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if tmpl.Labels["keep"] != "1" {
		t.Errorf("preexisting label dropped: %v", tmpl.Labels)
	}
}

func TestInjectAxisMLLabels_RejectsMissingJobID(t *testing.T) {
	mlj := validMLJob()
	delete(mlj.Labels, axislabels.JobIDLabel)
	err := InjectAxisMLLabels(&corev1.PodTemplateSpec{}, mlj, axisv1alpha1.RoleSpec{Name: "x"}, nil)
	if err == nil {
		t.Fatal("expected error for missing job-id")
	}
}

func TestInjectAxisMLLabels_RejectsMissingQuota(t *testing.T) {
	mlj := validMLJob()
	delete(mlj.Labels, axislabels.QuotaLabel)
	err := InjectAxisMLLabels(&corev1.PodTemplateSpec{}, mlj, axisv1alpha1.RoleSpec{Name: "x"}, nil)
	if err == nil {
		t.Fatal("expected error for missing quota label")
	}
}

func TestInjectAxisMLLabels_RejectsBlankSchedulingQuota(t *testing.T) {
	mlj := validMLJob()
	mlj.Spec.Scheduling.Quota = ""
	err := InjectAxisMLLabels(&corev1.PodTemplateSpec{}, mlj, axisv1alpha1.RoleSpec{Name: "x"}, nil)
	if err == nil {
		t.Fatal("expected error for blank scheduling.quota")
	}
}

func TestInjectAxisMLLabels_AppendsToExistingTolerations(t *testing.T) {
	mlj := validMLJob()
	tmpl := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Tolerations: []corev1.Toleration{{Key: "preexisting"}},
		},
	}
	if err := InjectAxisMLLabels(tmpl, mlj, axisv1alpha1.RoleSpec{Name: "x"}, nil); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(tmpl.Spec.Tolerations) != 2 {
		t.Errorf("tolerations should be appended; got %d", len(tmpl.Spec.Tolerations))
	}
}

func TestBuildContainer_AllFieldsCopied(t *testing.T) {
	role := axisv1alpha1.RoleSpec{
		Name: "worker",
		Template: axisv1alpha1.PodTemplateSubset{
			Image:           "img:tag",
			ImagePullPolicy: corev1.PullAlways,
			Command:         []string{"/bin/run"},
			Args:            []string{"--flag"},
			Env:             []corev1.EnvVar{{Name: "K", Value: "V"}},
			WorkingDir:      "/tmp",
		},
	}
	c := BuildContainer(role)
	if c.Name != "worker" || c.Image != "img:tag" || c.WorkingDir != "/tmp" {
		t.Errorf("container fields wrong: %+v", c)
	}
	if c.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("ImagePullPolicy = %s", c.ImagePullPolicy)
	}
	if len(c.Env) != 1 || c.Env[0].Name != "K" {
		t.Errorf("env not copied: %+v", c.Env)
	}
}

func TestOwnerRef_PointsToMLJob(t *testing.T) {
	mlj := validMLJob()
	ref := OwnerRef(mlj)
	if ref.Kind != "MLJob" {
		t.Errorf("Kind = %s", ref.Kind)
	}
	if ref.Name != mlj.Name || ref.UID != mlj.UID {
		t.Errorf("ref does not match MLJob: %+v", ref)
	}
	if ref.Controller == nil || !*ref.Controller {
		t.Error("Controller flag should be true")
	}
	if ref.BlockOwnerDeletion == nil || !*ref.BlockOwnerDeletion {
		t.Error("BlockOwnerDeletion should be true")
	}
}

func TestMapStatusResult_IsTerminal(t *testing.T) {
	cases := []struct {
		phase axisv1alpha1.MLJobPhase
		want  bool
	}{
		{axisv1alpha1.PhasePending, false},
		{axisv1alpha1.PhaseRunning, false},
		{axisv1alpha1.PhaseSucceeded, true},
		{axisv1alpha1.PhaseFailed, true},
		{"", false},
	}
	for _, tc := range cases {
		r := MapStatusResult{Phase: tc.phase}
		if got := r.IsTerminal(); got != tc.want {
			t.Errorf("phase=%s: IsTerminal()=%v want %v", tc.phase, got, tc.want)
		}
	}
}
