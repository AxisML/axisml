// Package handler defines the per-Kind handler interface and a process-
// global registry. New Kinds register themselves at init() time without
// touching the artifact service. See docs/system_design/artifacts.md §4.4.
package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/axisml/axisml/components/artifacts/internal/storage/oci"
)

// StorageKind enumerates the underlying storage backend a Kind uses.
type StorageKind string

const (
	StorageOCI StorageKind = "oci"
	StorageS3  StorageKind = "s3"
)

// Spec is the user-supplied Kind-specific declaration. Stored as JSON in
// the artifacts row; handlers parse it into their own typed struct.
type Spec map[string]any

// Artifact is the shape handlers see at runtime. Handlers must not mutate
// it — they only read fields and call backend operations.
type Artifact struct {
	Kind      string
	Namespace string // "tenants/<tenant>" for tenant-private; "system" for public
	Name      string
	Version   string
	Spec      Spec
	Digest    string // populated by VerifyComplete; empty before Ready
}

// Credentials is re-exported here so service callers don't need to depend
// on the storage package directly.
type Credentials = oci.Credentials

// CompleteClaim is the data the cli supplies when calling /complete.
type CompleteClaim struct {
	Digest string `json:"digest" binding:"required"`
}

// Handler is the per-Kind contract (design §4.4).
type Handler interface {
	Kind() string
	StorageKind() StorageKind
	BuildStorageURI(namespace, name, version string) string
	ValidateSpec(ctx context.Context, spec Spec) error
	InitiateUpload(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error)
	IssuePullCredentials(ctx context.Context, a Artifact, ttl time.Duration) (Credentials, error)
	VerifyComplete(ctx context.Context, a Artifact, claim CompleteClaim) (digest string, err error)
	GCBackend(ctx context.Context, a Artifact) error
}

var (
	regMu  sync.RWMutex
	regMap = map[string]Handler{}
)

// Register installs h in the process-global registry, keyed by Kind. Panics
// on duplicate registration so a misconfigured init() fails loudly.
func Register(h Handler) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, ok := regMap[h.Kind()]; ok {
		panic(fmt.Sprintf("handler: duplicate registration for kind %q", h.Kind()))
	}
	regMap[h.Kind()] = h
}

// Get returns the handler for the given Kind, or false if unregistered.
func Get(kind string) (Handler, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	h, ok := regMap[kind]
	return h, ok
}

// Kinds returns the sorted list of registered Kinds.
func Kinds() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(regMap))
	for k := range regMap {
		out = append(out, k)
	}
	return out
}

// reset is for tests only.
func reset() {
	regMu.Lock()
	defer regMu.Unlock()
	regMap = map[string]Handler{}
}
