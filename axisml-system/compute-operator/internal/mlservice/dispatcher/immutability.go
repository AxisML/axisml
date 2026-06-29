package dispatcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"
)

// immutableHashAnnotation stores the SHA-256 of the immutable spec subset
// recorded on the first successful Reconcile. Subsequent reconciles compare
// the live spec hash against this annotation; any drift is reported as
// "immutable field changed" per mlservice-operator.md §6.
const immutableHashAnnotation = "axisml.io/spec-immutable-hash"

// checkImmutability returns a non-empty message if the user mutated any spec
// field other than spec.roles[*].replicas after the baseline hash was
// recorded. It is read-only: the baseline is stamped separately by
// stampImmutabilityBaseline, only after Validate + Reconcile succeed, so a
// spec that fails validation does not get locked in as the immutable
// baseline (which would otherwise make the CR unrecoverable).
func (r *Reconciler) checkImmutability(mls *axisml.MLService) (string, error) {
	prev, ok := mls.Annotations[immutableHashAnnotation]
	if !ok {
		return "", nil
	}
	hash, err := immutableSpecHash(mls)
	if err != nil {
		return "", fmt.Errorf("hash immutable spec: %w", err)
	}
	if prev != hash {
		return "immutable spec field changed after creation; only spec.roles[*].replicas may be modified",
			nil
	}
	return "", nil
}

// stampImmutabilityBaseline records the current immutable spec hash on the
// CR if none has been recorded yet. Called only after a fully successful
// reconcile so that an invalid initial spec cannot lock the user out.
func (r *Reconciler) stampImmutabilityBaseline(ctx context.Context, mls *axisml.MLService) error {
	if _, ok := mls.Annotations[immutableHashAnnotation]; ok {
		return nil
	}
	hash, err := immutableSpecHash(mls)
	if err != nil {
		return fmt.Errorf("hash immutable spec: %w", err)
	}
	patched := mls.DeepCopy()
	if patched.Annotations == nil {
		patched.Annotations = map[string]string{}
	}
	patched.Annotations[immutableHashAnnotation] = hash
	if err := r.client.Patch(ctx, patched, client.MergeFrom(mls)); err != nil {
		return fmt.Errorf("stamp immutable hash: %w", err)
	}
	return nil
}

// immutableSpecHash hashes the immutable subset of the spec — i.e. every
// field except spec.roles[*].replicas. Implementation: deep-copy the spec,
// zero out the mutable fields, marshal canonically, hash.
//
// Backend.Config is a *runtime.RawExtension that stores the user-supplied
// JSON bytes verbatim. The apiserver may normalise those bytes on round-trip
// (whitespace, key ordering), so a literal byte hash would falsely flip
// "immutable changed" between create and read-back. We canonicalise by
// decoding the Raw bytes to a generic value and re-encoding — Go's
// json.Marshal sorts map keys, which gives us a stable representation.
func immutableSpecHash(mls *axisml.MLService) (string, error) {
	subset := mls.Spec.DeepCopy()
	for i := range subset.Roles {
		subset.Roles[i].Replicas = 0
	}
	if cfg := subset.Backend.Config; cfg != nil && len(cfg.Raw) > 0 {
		canon, err := canonicalizeJSON(cfg.Raw)
		if err != nil {
			return "", fmt.Errorf("canonicalize backend.config: %w", err)
		}
		subset.Backend.Config = &runtime.RawExtension{Raw: canon}
	}
	buf, err := json.Marshal(subset)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalizeJSON(raw []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
