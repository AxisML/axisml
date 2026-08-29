package extensions

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ResourceNode is one schedulable placement target. Allocatable is its total
// capacity; allocations are reported separately so Compute can reconcile live
// instances with durable desired reservations without double counting.
type ResourceNode struct {
	Name        string
	Labels      map[string]string
	Taints      []corev1.Taint
	Allocatable corev1.ResourceList
}

// ResourceAllocation is the request currently consumed by one live Pod or
// container. WorkloadID and Role allow Compute to match it to a durable
// MLRun/MLService reservation; unmatched allocations remain external usage.
type ResourceAllocation struct {
	NodeName   string
	WorkloadID string
	Role       string
	Replica    string
	Requests   corev1.ResourceList
}

// ResourceSnapshot is an internally consistent, point-in-time runtime view.
type ResourceSnapshot struct {
	ObservedAt    time.Time
	SourceVersion string
	Nodes         []ResourceNode
	Allocations   []ResourceAllocation
}

// ResourceInventory exposes capacity without admitting or creating workloads.
// Implementations must not mutate the runtime.
type ResourceInventory interface {
	Snapshot(ctx context.Context) (ResourceSnapshot, error)
}

// QuotaResolver returns the tenant's hard maximum for one ResourcePool.
// The same contract is backed by Tenant CRs on Kubernetes and the persistent
// standalone Tenant provider.
type QuotaResolver interface {
	ResolveQuota(ctx context.Context, tenant, pool string) (corev1.ResourceList, error)
}
