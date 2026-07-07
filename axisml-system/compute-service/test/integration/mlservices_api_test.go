//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"
)

func TestMLService_NotFound(t *testing.T) {
	ctx := context.Background()
	const ns = "services-nf-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/ghost", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/"+ns+"/mlservices/ghost/scale",
		map[string]any{"replicas": 2}, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/ghost/phase", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestMLService_ScaleValidation(t *testing.T) {
	// gte=0 fires at bind time, before the row lookup — any path works.
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/mlservices/whatever/scale",
		map[string]any{"replicas": -1}, nil)
	requireClientError(t, rr)
}

// TestMLService_Phase checks the lightweight phase probe: a freshly-created
// service is in the default Creating phase, and the projection carries the
// generation / observedGeneration sync signal but not the spec sub-tree.
func TestMLService_Phase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "services-phase-pool", "small-phase")
	const ns = "services-phase-ns"
	mustCreateNamespace(t, ctx, ns)

	body := map[string]any{
		"name":     "phase-svc",
		"poolName": "services-phase-pool",
		"unitName": "small-phase",
		"quota":    "axisml-default",
		"backend":  map[string]string{"name": "native", "engine": "deployment"},
		"roles": []map[string]any{
			{
				"name":     mlservicev1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{
					"image": "nginx:1.27",
					"ports": []map[string]any{
						{"name": "http", "containerPort": 8080, "protocol": string(corev1.ProtocolTCP)},
					},
				},
			},
		},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/phase-svc/phase", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "Creating", got["phase"])
	assert.Equal(t, "phase-svc", got["name"])
	assert.Contains(t, got, "generation")
	assert.Contains(t, got, "observedGeneration")
	assert.NotContains(t, got, "spec")
}

// TestMLService_BatchPhase covers the batch probe: ?names selects an explicit
// set (unresolved names omitted), empty ?names returns [], the cap is enforced,
// ?labelSelector filters, and ?kind narrows to one service kind like List does.
func TestMLService_BatchPhase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "services-batch-pool", "small-batch")
	const ns = "services-batch-ns"
	mustCreateNamespace(t, ctx, ns)

	build := func(name, kind string) map[string]any {
		return map[string]any{
			"name":     name,
			"kind":     kind,
			"labels":   map[string]string{"group": "ml"},
			"poolName": "services-batch-pool",
			"unitName": "small-batch",
			"quota":    "axisml-default",
			"backend":  map[string]string{"name": "native", "engine": "deployment"},
			"roles": []map[string]any{
				{
					"name":     mlservicev1alpha1.DefaultRoleName,
					"replicas": 1,
					"template": map[string]any{
						"image": "nginx:1.27",
						"ports": []map[string]any{
							{"name": "http", "containerPort": 8080, "protocol": string(corev1.ProtocolTCP)},
						},
					},
				},
			},
		}
	}
	// Two online services + one workspace, all labelled group=ml.
	for _, n := range []string{"batch-a", "batch-b"} {
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices",
			build(n, mlservicev1alpha1.ServiceKindService), nil)
		requireStatus(t, rr, http.StatusCreated)
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices",
		build("ws-a", mlservicev1alpha1.ServiceKindWorkspace), nil)
	requireStatus(t, rr, http.StatusCreated)

	type phaseList struct {
		Items []map[string]any `json:"items"`
		Count int              `json:"count"`
		Total int64            `json:"total"`
	}
	names := func(items []map[string]any) map[string]bool {
		out := map[string]bool{}
		for _, it := range items {
			name, _ := it["name"].(string)
			out[name] = true
			assert.Equal(t, "Creating", it["phase"])
			assert.Contains(t, it, "observedGeneration")
			assert.NotContains(t, it, "spec")
		}
		return out
	}

	// ?names selects an explicit set; the unresolved "ghost" is omitted.
	var byNames phaseList
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlservices/phases?names=batch-a,ghost,batch-b", nil, &byNames)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, 2, byNames.Count)
	assert.Equal(t, map[string]bool{"batch-a": true, "batch-b": true}, names(byNames.Items))

	// An explicit empty ?names returns an empty set, NOT the whole namespace.
	var empty phaseList
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/phases?names=", nil, &empty)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, 0, empty.Count)
	assert.Empty(t, empty.Items)

	// More names than the cap → 400 validation.
	tooMany := make([]string, 201)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("n%d", i)
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlservices/phases?names="+strings.Join(tooMany, ","), nil, nil)
	requireStatus(t, rr, http.StatusBadRequest)

	// ?labelSelector alone spans all three kinds.
	var bySelector phaseList
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlservices/phases?labelSelector="+url.QueryEscape("group=ml"), nil, &bySelector)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(3), bySelector.Total)

	// ?kind=service narrows to the two online services, excluding the workspace.
	var byKind phaseList
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlservices/phases?kind="+mlservicev1alpha1.ServiceKindService, nil, &byKind)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(2), byKind.Total)
	assert.Equal(t, map[string]bool{"batch-a": true, "batch-b": true}, names(byKind.Items))

	// The static /phases route must not shadow the single-get param route.
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/batch-a", nil, nil)
	requireStatus(t, rr, http.StatusOK)
}
