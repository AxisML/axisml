package handler

import (
	"testing"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

func TestStubsValidateReturnDocPointer(t *testing.T) {
	h := &notImplementedHandler{key: Key{Backend: "kserve", Engine: "inference"}}
	v := h.Validate(&axisml.MLServiceSpec{})
	if v.OK() {
		t.Fatal("expected stub Validate to fail")
	}
	found := false
	for _, e := range v.Errors {
		if containsAll(e, "kserve", "inference", "compute-operator.md", "§4") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error to reference design doc and key tuple; got %v", v.Errors)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !indexOf(s, sub) {
			return false
		}
	}
	return true
}

func indexOf(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
