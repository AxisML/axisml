// Package nativejob implements the (native, job) handler from
// design §8.1: a single-role MLJob is rendered as one batch/v1.Job.
// All Pods carry koord-scheduler + the five mandatory labels so the
// ElasticQuota plugin accounts for them; gang scheduling is NOT used
// (that's the (native, podgroup) handler's job).
package nativejob

import (
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axishandler "axisml.io/operators/mljob/internal/handler"
)

const (
	BackendName   = "native"
	BackendEngine = "job"

	roleName = "worker"
)

// Handler implements axishandler.Handler.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Key() axishandler.Key {
	return axishandler.Key{Backend: BackendName, Engine: BackendEngine}
}

func (h *Handler) WatchTargets() []client.Object {
	// The Job's Pods are owned by the Job; we only watch Job. This keeps
	// the MapStatus path on the path Job→Pods that batch/v1 already
	// aggregates, and avoids redundant reconciles per Pod event.
	return []client.Object{&batchv1.Job{}}
}

type backendConfig struct {
	CompletionMode   batchv1.CompletionMode    `json:"completionMode,omitempty"`
	PodFailurePolicy *batchv1.PodFailurePolicy `json:"podFailurePolicy,omitempty"`
}

func jobName(mljobName string) string { return mljobName }
