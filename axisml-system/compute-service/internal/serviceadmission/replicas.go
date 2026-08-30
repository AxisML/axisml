// Package serviceadmission owns the replica-vector representation shared by
// MLService admission, runtime rendering and API projections. Vectors are
// indexed by immutable spec.roles order; this avoids role-name ambiguity while
// keeping the persisted jsonb compact.
package serviceadmission

import (
	"encoding/json"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// Desired returns the non-negative desired replica vector from a service spec.
func Desired(spec mlservicev1alpha1.MLServiceSpec) []int32 {
	out := make([]int32, len(spec.Roles))
	for i := range spec.Roles {
		if spec.Roles[i].Replicas > 0 {
			out[i] = spec.Roles[i].Replicas
		}
	}
	return out
}

// Zero returns a zeroed vector aligned with spec.roles.
func Zero(spec mlservicev1alpha1.MLServiceSpec) []int32 {
	return make([]int32, len(spec.Roles))
}

// Decode normalizes a persisted vector to roleCount entries. A missing legacy
// value falls back to fallback, allowing pre-admission in-memory fixtures to
// retain their already-dispatched desired replicas. Persisted [] remains an
// authoritative zero vector.
func Decode(raw []byte, roleCount int, fallback []int32) []int32 {
	var decoded []int32
	legacyMissing := len(raw) == 0
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &decoded)
	}
	if legacyMissing && len(fallback) > 0 {
		decoded = append([]int32(nil), fallback...)
	}
	out := make([]int32, roleCount)
	for i := range out {
		if i < len(decoded) && decoded[i] > 0 {
			out[i] = decoded[i]
		}
	}
	return out
}

// Encode returns the canonical persisted JSON representation.
func Encode(replicas []int32) []byte {
	b, _ := json.Marshal(replicas)
	return b
}

// Equal reports vector equality after callers have normalized both sides.
func Equal(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Primary returns the role[0] replica count used by today's Platform API.
func Primary(replicas []int32) int32 {
	if len(replicas) == 0 {
		return 0
	}
	return replicas[0]
}

// ClampToDesired immediately releases admission above a reduced desired
// vector. Capacity growth is handled separately by the admission controller.
func ClampToDesired(admitted, desired []int32) []int32 {
	out := make([]int32, len(desired))
	for i := range desired {
		if i < len(admitted) && admitted[i] < desired[i] {
			out[i] = admitted[i]
		} else {
			out[i] = desired[i]
		}
	}
	return out
}

// Next returns the next atomic admission step. Before a service has any
// admitted replica, one replica of every non-empty role is admitted together as
// the minimum serving set. After that, replicas grow one at a time in immutable
// role order. The returned indices are the roles whose replica count increased.
func Next(admitted, desired []int32) (next []int32, increased []int) {
	next = append([]int32(nil), admitted...)
	if len(next) < len(desired) {
		next = append(next, make([]int32, len(desired)-len(next))...)
	}
	hasAny := false
	for _, n := range admitted {
		if n > 0 {
			hasAny = true
			break
		}
	}
	if !hasAny {
		for i := range desired {
			if desired[i] > 0 {
				next[i] = 1
				increased = append(increased, i)
			}
		}
		return next, increased
	}
	for i := range desired {
		if next[i] < desired[i] {
			next[i]++
			return next, []int{i}
		}
	}
	return next, nil
}

// Apply returns a spec whose role replicas are limited to the admitted vector.
func Apply(spec mlservicev1alpha1.MLServiceSpec, admitted []int32) mlservicev1alpha1.MLServiceSpec {
	out := spec
	out.Roles = append([]mlservicev1alpha1.RoleSpec(nil), spec.Roles...)
	for i := range out.Roles {
		out.Roles[i].Replicas = 0
		if i < len(admitted) && admitted[i] > 0 {
			out.Roles[i].Replicas = admitted[i]
		}
	}
	return out
}
