package handler

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// withFreshRegistry runs fn against a clean factories map and restores the
// previous state when it returns. Stub registration uses init-time globals,
// so a per-test isolated map keeps these tests order-independent.
func withFreshRegistry(t *testing.T, fn func()) {
	t.Helper()
	registryMu.Lock()
	saved := factories
	factories = map[Key]Factory{}
	registryMu.Unlock()

	defer func() {
		registryMu.Lock()
		factories = saved
		registryMu.Unlock()
	}()
	fn()
}

func TestKey_String(t *testing.T) {
	got := Key{Backend: "native", Engine: "deployment"}.String()
	if got != "native/deployment" {
		t.Errorf("Key.String() = %q", got)
	}
}

func TestValidation_OK(t *testing.T) {
	if !(Validation{}).OK() {
		t.Error("empty Validation should be OK")
	}
	v := Validation{Errors: []string{"x"}}
	if v.OK() {
		t.Error("Validation with errors should not be OK")
	}
	if !(Validation{Warnings: []string{"x"}}).OK() {
		t.Error("Warnings alone should still be OK")
	}
}

func TestRegister_PanicsOnDuplicate(t *testing.T) {
	withFreshRegistry(t, func() {
		k := Key{Backend: "a", Engine: "b"}
		Register(k, func(_ manager.Manager) (Handler, error) { return nil, nil })

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on duplicate registration")
			}
		}()
		Register(k, func(_ manager.Manager) (Handler, error) { return nil, nil })
	})
}

func TestKeys_ReturnsAllRegistered(t *testing.T) {
	withFreshRegistry(t, func() {
		Register(Key{Backend: "a", Engine: "1"}, func(_ manager.Manager) (Handler, error) { return nil, nil })
		Register(Key{Backend: "b", Engine: "2"}, func(_ manager.Manager) (Handler, error) { return nil, nil })
		got := Keys()
		if len(got) != 2 {
			t.Fatalf("len = %d; want 2", len(got))
		}
	})
}

func TestRegisterStubs_RegistersTwoKserveKeys(t *testing.T) {
	withFreshRegistry(t, func() {
		RegisterStubs()
		got := Keys()
		if len(got) != 2 {
			t.Fatalf("len = %d; want 2 stub keys", len(got))
		}
		seenInference, seenLLM := false, false
		for _, k := range got {
			if k.Backend != "kserve" {
				t.Errorf("non-kserve key registered: %s", k)
			}
			switch k.Engine {
			case "inference":
				seenInference = true
			case "llminference":
				seenLLM = true
			}
		}
		if !seenInference || !seenLLM {
			t.Errorf("missing stub keys: got %v", got)
		}
	})
}

func TestNotImplementedHandler_Behavior(t *testing.T) {
	h := &notImplementedHandler{key: Key{Backend: "kserve", Engine: "inference"}}
	if h.Key() != (Key{Backend: "kserve", Engine: "inference"}) {
		t.Errorf("Key() = %s", h.Key())
	}
	if _, err := h.Reconcile(context.Background(), &axisml.MLService{}); err != nil {
		t.Errorf("Reconcile should be a no-op; got err=%v", err)
	}
	if err := h.Cleanup(context.Background(), &axisml.MLService{}); err != nil {
		t.Errorf("Cleanup should be a no-op; got err=%v", err)
	}
	su := h.MapStatus(Snapshot{})
	if su.Phase != axisml.PhaseFailed {
		t.Errorf("MapStatus phase = %s; want Failed", su.Phase)
	}
	if got := h.WatchTargets(); got != nil {
		t.Errorf("WatchTargets should be nil; got %v", got)
	}
	if got := h.RequiredRBAC(); got != nil {
		t.Errorf("RequiredRBAC should be nil; got %v", got)
	}
}
