package quota

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func rl(cpu, gpu string) v1.ResourceList {
	out := v1.ResourceList{v1.ResourceCPU: resource.MustParse(cpu)}
	if gpu != "" {
		out["nvidia.com/gpu"] = resource.MustParse(gpu)
	}
	return out
}

func TestCacheSumsRequestsAndDedupes(t *testing.T) {
	c := New()
	k := Key("team-a", "team-a-default-q1")

	c.AddPod(k, "pod-1", rl("2", "1"))
	c.AddPod(k, "pod-2", rl("3", "1"))
	// Re-adding the same pod key (e.g. Reserve then informer Add) must not double-count.
	c.AddPod(k, "pod-1", rl("2", "1"))

	used := c.Used(k)
	if got := used[v1.ResourceCPU]; got.Cmp(resource.MustParse("5")) != 0 {
		t.Fatalf("cpu used = %s; want 5", got.String())
	}
	if got := used["nvidia.com/gpu"]; got.Cmp(resource.MustParse("2")) != 0 {
		t.Fatalf("gpu used = %s; want 2", got.String())
	}
}

func TestCacheRemoveAndEmpty(t *testing.T) {
	c := New()
	k := Key("team-a", "q1")

	c.AddPod(k, "pod-1", rl("2", ""))
	c.RemovePod(k, "pod-1")

	if used := c.Used(k); len(used) != 0 {
		t.Fatalf("used after remove = %v; want empty", used)
	}
	// Removing an unknown pod is a no-op.
	c.RemovePod(k, "missing")
}

func TestCacheReAddSameUIDAdjustsNotAccumulates(t *testing.T) {
	c := New()
	k := Key("team-a", "q1")

	c.AddPod(k, "pod-1", rl("2", "1"))
	// Re-add the same UID with a DIFFERENT request (e.g. informer replay after a
	// spec change): the total must reflect the new value, not the sum.
	c.AddPod(k, "pod-1", rl("5", ""))

	used := c.Used(k)
	if got := used[v1.ResourceCPU]; got.Cmp(resource.MustParse("5")) != 0 {
		t.Fatalf("cpu used = %s; want 5 (adjusted, not accumulated)", got.String())
	}
	// The gpu from the first add must be gone (second req had none).
	if _, ok := used["nvidia.com/gpu"]; ok {
		t.Fatalf("gpu still present after re-add dropped it: %v", used)
	}
}

func TestTerminal(t *testing.T) {
	for phase, want := range map[v1.PodPhase]bool{
		v1.PodSucceeded: true,
		v1.PodFailed:    true,
		v1.PodRunning:   false,
		v1.PodPending:   false,
	} {
		p := &v1.Pod{Status: v1.PodStatus{Phase: phase}}
		if got := Terminal(p); got != want {
			t.Errorf("Terminal(%s) = %v; want %v", phase, got, want)
		}
	}
}

func TestKeyShape(t *testing.T) {
	if got := Key("ns", "name"); got != "ns/name" {
		t.Fatalf("Key = %q; want ns/name", got)
	}
}
