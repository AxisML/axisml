package dispatcher

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

func TestImmutableSpecHash_StableOnReplicaChange(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke"},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
			Scheduling: axisml.Scheduling{Quota: "q"},
			ModelRef:   axisml.ModelRef{Name: "m", Version: "v1"},
			Roles: []axisml.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: axisml.PodTemplate{Image: "nginx:1.27"},
			}},
		},
	}
	h1, err := immutableSpecHash(mls)
	if err != nil {
		t.Fatal(err)
	}

	mls.Spec.Roles[0].Replicas = 5
	h2, err := immutableSpecHash(mls)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash changed on replicas-only mutation: %s vs %s", h1, h2)
	}
}

func TestImmutableSpecHash_ChangesOnImageMutation(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke"},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
			Scheduling: axisml.Scheduling{Quota: "q"},
			ModelRef:   axisml.ModelRef{Name: "m", Version: "v1"},
			Roles: []axisml.RoleSpec{{
				Name:     "predictor",
				Replicas: 1,
				Template: axisml.PodTemplate{Image: "nginx:1.27"},
			}},
		},
	}
	h1, err := immutableSpecHash(mls)
	if err != nil {
		t.Fatal(err)
	}

	mls.Spec.Roles[0].Template.Image = "nginx:1.28"
	h2, err := immutableSpecHash(mls)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("hash unchanged after image mutation; immutable detection would miss it")
	}
}

func TestImmutableSpecHash_ChangesOnBackendMutation(t *testing.T) {
	mls := &axisml.MLService{
		Spec: axisml.MLServiceSpec{
			Backend: axisml.Backend{Name: "native", Engine: "deployment"},
			Roles:   []axisml.RoleSpec{{Name: "predictor", Replicas: 1}},
		},
	}
	h1, _ := immutableSpecHash(mls)
	mls.Spec.Backend.Engine = "statefulset"
	h2, _ := immutableSpecHash(mls)
	if h1 == h2 {
		t.Error("hash unchanged after backend.engine mutation")
	}
}

// TestImmutableSpecHash_CanonicalizesBackendConfig guards against false
// "immutable changed" detections caused by apiserver byte-level normalisation
// of Backend.Config. Two specs whose only difference is the JSON-encoding
// (whitespace, key order) of Backend.Config must hash the same.
func TestImmutableSpecHash_CanonicalizesBackendConfig(t *testing.T) {
	build := func(raw string) *axisml.MLService {
		return &axisml.MLService{
			Spec: axisml.MLServiceSpec{
				Backend: axisml.Backend{
					Name:   "native",
					Engine: "deployment",
					Config: &runtime.RawExtension{Raw: []byte(raw)},
				},
				Roles: []axisml.RoleSpec{{Name: "predictor", Replicas: 1}},
			},
		}
	}
	h1, err := immutableSpecHash(build(`{"a":1,"b":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	h2, err := immutableSpecHash(build("{\n  \"b\": \"x\",\n  \"a\": 1\n}"))
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Errorf("hash differed across whitespace/key-order variants: %s vs %s", h1, h2)
	}
}
