package resourcepool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func TestLabelSelectorFrom(t *testing.T) {
	t.Run("empty is match-all", func(t *testing.T) {
		sel, err := labelSelectorFrom("")
		require.NoError(t, err)
		assert.True(t, sel.Empty())
	})

	t.Run("valid selector parses", func(t *testing.T) {
		sel, err := labelSelectorFrom("team=ml,tier in (gpu)")
		require.NoError(t, err)
		assert.False(t, sel.Empty())
	})

	t.Run("invalid selector errors", func(t *testing.T) {
		_, err := labelSelectorFrom("!!bad==")
		assert.Error(t, err)
	})
}

func TestFirstDuplicateUnit(t *testing.T) {
	t.Run("no duplicates", func(t *testing.T) {
		dup, name := firstDuplicateUnit([]srv.CreateResourceUnitRequest{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		})
		assert.False(t, dup)
		assert.Empty(t, name)
	})

	t.Run("empty slice", func(t *testing.T) {
		dup, name := firstDuplicateUnit(nil)
		assert.False(t, dup)
		assert.Empty(t, name)
	})

	t.Run("reports first duplicate", func(t *testing.T) {
		dup, name := firstDuplicateUnit([]srv.CreateResourceUnitRequest{
			{Name: "a"}, {Name: "b"}, {Name: "a"},
		})
		assert.True(t, dup)
		assert.Equal(t, "a", name)
	})
}

func TestApplyPoolPatch(t *testing.T) {
	t.Run("updates fields and stamps user", func(t *testing.T) {
		pool := &axismlv1alpha1.ResourcePool{}
		pool.Annotations = map[string]string{
			srv.DescriptionAnnotation:    "old-desc",
			srv.LastModifiedByAnnotation: "old-user",
			"team":                       "keep-me",
		}
		desc := "new-desc"
		req := srv.PatchResourcePoolRequest{
			Description:  &desc,
			NodeSelector: map[string]string{"gpu": "true"},
			Capacity:     corev1.ResourceList{"cpu": resource.MustParse("8")},
			Labels:       map[string]string{"tier": "prod"},
			Annotations:  map[string]string{"team": "ml"},
		}
		applyPoolPatch(pool, req, "alice")

		assert.Equal(t, map[string]string{"gpu": "true"}, pool.Spec.NodeSelector)
		assert.Equal(t, "8", pool.Spec.Capacity.Cpu().String())
		assert.Equal(t, map[string]string{"tier": "prod"}, pool.Labels)
		// User annotations replace user-visible keys; reserved ones are preserved.
		assert.Equal(t, "ml", pool.Annotations["team"])
		assert.Equal(t, "new-desc", pool.Annotations[srv.DescriptionAnnotation])
		assert.Equal(t, "alice", pool.Annotations[srv.LastModifiedByAnnotation])
	})

	t.Run("nil fields leave existing untouched, nil annotations init", func(t *testing.T) {
		pool := &axismlv1alpha1.ResourcePool{}
		pool.Spec.NodeSelector = map[string]string{"keep": "yes"}
		applyPoolPatch(pool, srv.PatchResourcePoolRequest{}, "")
		// Nothing supplied and no user: node selector kept, annotations initialised empty.
		assert.Equal(t, map[string]string{"keep": "yes"}, pool.Spec.NodeSelector)
		assert.NotNil(t, pool.Annotations)
		assert.Empty(t, pool.Annotations)
	})

	t.Run("empty-string description clears via replace path", func(t *testing.T) {
		pool := &axismlv1alpha1.ResourcePool{}
		empty := ""
		applyPoolPatch(pool, srv.PatchResourcePoolRequest{Description: &empty}, "")
		assert.Equal(t, "", pool.Annotations[srv.DescriptionAnnotation])
	})
}

func TestApplyUnitPatch(t *testing.T) {
	t.Run("replaces requests, limits, selector and preserves description", func(t *testing.T) {
		u := &axismlv1alpha1.ResourceUnit{
			Annotations: map[string]string{srv.DescriptionAnnotation: "keep-desc", "extra": "gone"},
		}
		req := srv.PatchResourceUnitRequest{
			Requests:     corev1.ResourceList{"cpu": resource.MustParse("2")},
			Limits:       corev1.ResourceList{"cpu": resource.MustParse("4")},
			NodeSelector: map[string]string{"zone": "a"},
			Annotations:  map[string]string{"new": "v"},
		}
		applyUnitPatch(u, req)

		assert.Equal(t, "2", u.Requests.Cpu().String())
		assert.Equal(t, "4", u.Limits.Cpu().String())
		assert.Equal(t, map[string]string{"zone": "a"}, u.NodeSelector)
		assert.Equal(t, "v", u.Annotations["new"])
		assert.Equal(t, "keep-desc", u.Annotations[srv.DescriptionAnnotation])
		// A non-reserved annotation not present in the patch is dropped.
		_, ok := u.Annotations["extra"]
		assert.False(t, ok)
	})

	t.Run("description patch on unit with nil annotations", func(t *testing.T) {
		u := &axismlv1alpha1.ResourceUnit{}
		desc := "hello"
		applyUnitPatch(u, srv.PatchResourceUnitRequest{Description: &desc})
		assert.Equal(t, "hello", u.Annotations[srv.DescriptionAnnotation])
	})

	t.Run("empty request is a no-op", func(t *testing.T) {
		u := &axismlv1alpha1.ResourceUnit{
			Requests: corev1.ResourceList{"cpu": resource.MustParse("1")},
		}
		applyUnitPatch(u, srv.PatchResourceUnitRequest{})
		assert.Equal(t, "1", u.Requests.Cpu().String())
		assert.Nil(t, u.Annotations)
	})
}
