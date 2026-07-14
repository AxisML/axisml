package workloadname

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamingContract(t *testing.T) {
	obj := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "hello-world"}}
	Annotate(obj, "team-a", "hello-world", true)
	if got := Workload(obj); got != "team-a-96c2886c-hello-world" {
		t.Fatalf("Workload() = %q", got)
	}
	if got := Role(obj, "worker"); got != "team-a-96c2886c-hello-world-worker" {
		t.Fatalf("Role() = %q", got)
	}
	if got := Related(obj, "team-a", "hello-green"); got != "team-a-96c2886c-hello-green" {
		t.Fatalf("Related() = %q", got)
	}
}

func TestBaseTenantPrefixSeparatesAmbiguousTuples(t *testing.T) {
	first := Base("team-a", "hello", true)
	second := Base("team", "a-hello", true)
	if first == second {
		t.Fatalf("ambiguous tenant/workload tuples both produced %q", first)
	}
}

func TestBaseWithoutTenantPrefix(t *testing.T) {
	if got := Base("team-a", "hello-world", false); got != "hello-world" {
		t.Fatalf("Base() = %q", got)
	}
}

func TestRoleKeepsRoleAndRuntimeSuffixRoom(t *testing.T) {
	obj := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 63)}}
	got := Role(obj, "worker")
	if len(got) > 57 || !strings.HasSuffix(got, "-worker") {
		t.Fatalf("Role() = %q", got)
	}
}
