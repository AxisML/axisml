package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var dnsLabelInvalid = regexp.MustCompile(`[^a-z0-9-]`)

// RandomNamespace creates a Namespace with a deterministic-but-unique name
// derived from the test name and a random suffix, then registers a t.Cleanup
// hook to delete it. Returns the namespace name.
//
// Used by L1 integration tests against envtest's embedded apiserver, where
// Namespace deletion is instantaneous.
func RandomNamespace(t *testing.T, ctx context.Context, c client.Client, prefix string) string {
	t.Helper()
	name := UniqueNamespaceName(t.Name(), prefix)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := c.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace %q: %v", name, err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var got corev1.Namespace
		err := c.Get(ctx, types.NamespacedName{Name: name}, &got)
		switch {
		case apierrors.IsNotFound(err):
			return
		case err != nil:
			t.Logf("cleanup get namespace %q: %v", name, err)
			return
		}
		if err := c.Delete(ctx, &got); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("cleanup delete namespace %q: %v", name, err)
		}
	})
	return name
}

// UniqueNamespaceName builds a DNS-label-safe namespace name from a test name,
// a prefix, and a 6-hex-char random suffix. Length capped at 63 chars.
func UniqueNamespaceName(testName, prefix string) string {
	var suffix [3]byte
	_, _ = rand.Read(suffix[:])
	slug := dnsLabelInvalid.ReplaceAllString(strings.ToLower(testName), "-")
	slug = strings.Trim(slug, "-")
	const maxSlug = 40
	if len(slug) > maxSlug {
		slug = slug[:maxSlug]
	}
	name := prefix + slug + "-" + hex.EncodeToString(suffix[:])
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}
