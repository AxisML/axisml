package server_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func TestPoolToAPI(t *testing.T) {
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cr := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "gpu",
			ResourceVersion:   "42",
			CreationTimestamp: metav1.NewTime(created),
			Labels:            map[string]string{"tier": "prod"},
			Annotations: map[string]string{
				srv.DescriptionAnnotation:    "a pool",
				srv.LastModifiedByAnnotation: "alice",
				srv.QuotasAnnotation:         "should-be-stripped",
				"team":                       "ml",
			},
		},
		Spec: axismlv1alpha1.ResourcePoolSpec{
			NodeSelector: map[string]string{"gpu": "true"},
			Tolerations:  []corev1.Toleration{{Key: "dedicated"}},
			Units: []axismlv1alpha1.ResourceUnit{{
				Name:        "small",
				Annotations: map[string]string{srv.DescriptionAnnotation: "unit desc", "x": "y"},
				Requests:    corev1.ResourceList{"cpu": resource.MustParse("1")},
				Limits:      corev1.ResourceList{"cpu": resource.MustParse("2")},
			}},
		},
	}

	dto := srv.PoolToAPI(cr)
	assert.Equal(t, "gpu", dto.Name)
	assert.Equal(t, "a pool", dto.Description)
	assert.Equal(t, "42", dto.ResourceVersion)
	assert.Equal(t, created, dto.CreatedAt)
	assert.Equal(t, map[string]string{"gpu": "true"}, dto.NodeSelector)
	require.Len(t, dto.Units, 1)
	assert.Equal(t, "small", dto.Units[0].Name)
	assert.Equal(t, "unit desc", dto.Units[0].Description)
	// Reserved annotations are stripped; user annotations survive.
	assert.Equal(t, map[string]string{"team": "ml"}, dto.Annotations)
	assert.Equal(t, map[string]string{"x": "y"}, dto.Units[0].Annotations)
}

func TestPoolToAPI_OnlyReservedAnnotationsStripToNil(t *testing.T) {
	cr := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "p",
			Annotations: map[string]string{srv.DescriptionAnnotation: "d"},
		},
	}
	dto := srv.PoolToAPI(cr)
	assert.Equal(t, "d", dto.Description)
	assert.Nil(t, dto.Annotations)
	assert.Empty(t, dto.Units)
}

func TestUnitToAPI(t *testing.T) {
	u := axismlv1alpha1.ResourceUnit{
		Name:         "u",
		NodeSelector: map[string]string{"z": "1"},
		Annotations:  map[string]string{srv.DescriptionAnnotation: "desc"},
	}
	dto := srv.UnitToAPI(u)
	assert.Equal(t, "u", dto.Name)
	assert.Equal(t, "desc", dto.Description)
	assert.Equal(t, map[string]string{"z": "1"}, dto.NodeSelector)
	assert.Nil(t, dto.Annotations)
}

func TestAPIToPool(t *testing.T) {
	req := srv.CreateResourcePoolRequest{
		Name:         "p1",
		Description:  "desc",
		NodeSelector: map[string]string{"gpu": "true"},
		Tolerations:  []corev1.Toleration{{Key: "k"}},
		Labels:       map[string]string{"tier": "prod"},
		Annotations:  map[string]string{"team": "ml"},
		Units: []srv.CreateResourceUnitRequest{{
			Name:        "small",
			Description: "unit",
			Requests:    corev1.ResourceList{"cpu": resource.MustParse("1")},
		}},
	}
	pool := srv.APIToPool(req, "alice")

	assert.Equal(t, "p1", pool.Name)
	assert.Equal(t, map[string]string{"tier": "prod"}, pool.Labels)
	assert.Equal(t, "desc", pool.Annotations[srv.DescriptionAnnotation])
	assert.Equal(t, "alice", pool.Annotations[srv.LastModifiedByAnnotation])
	assert.Equal(t, "ml", pool.Annotations["team"])
	require.Len(t, pool.Spec.Units, 1)
	assert.Equal(t, "small", pool.Spec.Units[0].Name)
	assert.Equal(t, "unit", pool.Spec.Units[0].Annotations[srv.DescriptionAnnotation])
	// The unit carries no last-modified-by (only the pool-level call passes user).
	_, ok := pool.Spec.Units[0].Annotations[srv.LastModifiedByAnnotation]
	assert.False(t, ok)

	// Deep copy: mutating the request must not touch the built CR.
	req.NodeSelector["gpu"] = "mutated"
	assert.Equal(t, "true", pool.Spec.NodeSelector["gpu"])
}

func TestAPIToPool_NoAnnotationsYieldsNil(t *testing.T) {
	pool := srv.APIToPool(srv.CreateResourcePoolRequest{Name: "p"}, "")
	assert.Nil(t, pool.Annotations)
	assert.Nil(t, pool.Labels)
	assert.Nil(t, pool.Spec.Tolerations)
}

func TestAPIToUnit(t *testing.T) {
	req := srv.CreateResourceUnitRequest{
		Name:         "u",
		Description:  "d",
		Requests:     corev1.ResourceList{"cpu": resource.MustParse("1")},
		NodeSelector: map[string]string{"z": "1"},
		Annotations:  map[string]string{"k": "v"},
	}
	u := srv.APIToUnit(req)
	assert.Equal(t, "u", u.Name)
	assert.Equal(t, "d", u.Annotations[srv.DescriptionAnnotation])
	assert.Equal(t, "v", u.Annotations["k"])
	assert.Equal(t, map[string]string{"z": "1"}, u.NodeSelector)
}
