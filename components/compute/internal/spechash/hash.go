package spechash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Compute serializes spec via encoding/json (which sorts map keys and
// emits struct fields in declaration order — both deterministic) and
// returns the SHA-256 hex digest.
func Compute(spec any) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
