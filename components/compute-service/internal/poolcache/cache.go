// Package poolcache wraps the controller-runtime Informer cache for the
// ResourcePool CRD (cluster-scoped). Compute consumes this cache at
// Job/Service create time to expand (poolName, unitName) into the
// nodeSelector / tolerations / resources snapshot that lands in spec
// jsonb — there is no PG mirror of ResourcePool. See design §5.4.
package poolcache

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

// Reader exposes the read-only lookups needed by job/service. Backed by
// a controller-runtime client.Client whose cache the parent manager has
// already wired (so Get is served from the local Informer cache).
type Reader struct {
	c client.Reader
}

// New returns a Reader that calls c.Get to satisfy lookups.
func New(c client.Reader) *Reader { return &Reader{c: c} }

// Expanded is the snapshot pulled out of the CR cache at create time.
type Expanded struct {
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
	Requests     corev1.ResourceList
	Limits       corev1.ResourceList
}

// Resolve looks up (poolName, unitName) and returns the expanded snapshot
// per design §5.4 merge rules (pool nodeSelector keys win; pool tolerations
// pass through verbatim; unit requests/limits go to the role template).
func (r *Reader) Resolve(ctx context.Context, poolName, unitName string) (*Expanded, error) {
	if poolName == "" || unitName == "" {
		return nil, apperrors.New(apperrors.CodeValidation,
			"poolName and unitName are required")
	}
	var pool axismlv1alpha1.ResourcePool
	if err := r.c.Get(ctx, types.NamespacedName{Name: poolName}, &pool); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperrors.Newf(apperrors.CodeValidation,
				"resource pool %q not found", poolName)
		}
		return nil, apperrors.Wrap(apperrors.CodeUnavailable,
			fmt.Sprintf("read ResourcePool %q", poolName), err)
	}

	for i := range pool.Spec.Units {
		u := &pool.Spec.Units[i]
		if u.Name != unitName {
			continue
		}
		return &Expanded{
			NodeSelector: mergeNodeSelector(pool.Spec.NodeSelector, u.NodeSelector),
			Tolerations:  pool.Spec.Tolerations,
			Requests:     u.Requests,
			Limits:       u.Limits,
		}, nil
	}
	return nil, apperrors.Newf(apperrors.CodeValidation,
		"resource unit %q not found in pool %q", unitName, poolName)
}

// mergeNodeSelector applies the design merge rule: pool keys are
// preserved, unit-only keys fill in gaps the pool didn't declare.
func mergeNodeSelector(poolSel, unitSel map[string]string) map[string]string {
	if len(poolSel) == 0 && len(unitSel) == 0 {
		return nil
	}
	out := make(map[string]string, len(poolSel)+len(unitSel))
	for k, v := range poolSel {
		out[k] = v
	}
	for k, v := range unitSel {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

// BuildResources converts the unit's ResourceList into the K8s
// ResourceRequirements shape expected by Pod templates.
func BuildResources(req, lim corev1.ResourceList) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if len(req) > 0 {
		rr.Requests = req
	}
	if len(lim) > 0 {
		rr.Limits = lim
	}
	return rr
}
