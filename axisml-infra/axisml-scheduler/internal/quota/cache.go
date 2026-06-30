// Package quota holds the label-keyed ElasticQuota usage accounting shared by
// the axisml-scheduler ElasticScheduling plugin and controller.
//
// Unlike upstream scheduler-plugins CapacityScheduling — which associates a Pod
// to an ElasticQuota by namespace — AxisML binds by the Pod label
// scheduling.axisml.io/quota, so a single namespace (= tenant) can carry many
// quotas (one per pool). The cache is keyed by (namespace, quota-name).
package quota

import (
	"sync"

	v1 "k8s.io/api/core/v1"
	resourcehelper "k8s.io/component-helpers/resource"
)

// LabelQuota binds a Pod to an ElasticQuota in the same namespace by name.
const LabelQuota = "scheduling.axisml.io/quota"

// Key is the cache key for an ElasticQuota: its namespace and name.
func Key(namespace, name string) string { return namespace + "/" + name }

// Terminal reports whether a Pod has reached a terminal phase and therefore no
// longer consumes quota.
func Terminal(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
}

// PodRequest returns the Pod's total resource request (init/sidecar/overhead
// accounted by the shared helper). Used by both the plugin and the controller
// so admission and reported usage compute the same number.
func PodRequest(pod *v1.Pod) v1.ResourceList {
	return resourcehelper.PodRequests(pod, resourcehelper.PodResourcesOptions{})
}

// Cache tracks the resource requests of the Pods accounted to each quota.
// It maintains a running total per quota so Used() is O(1). Safe for concurrent
// use.
type Cache struct {
	mu    sync.RWMutex
	infos map[string]*info
}

type info struct {
	// pods maps a stable pod key (UID) to its recorded request, so the same
	// pod observed twice (Reserve + Pod informer Add) is not double-counted and
	// removal can subtract the exact contribution.
	pods map[string]v1.ResourceList
	// total is the running sum of pods' requests, kept in sync on every mutation
	// so Used() is a copy, not a re-summation.
	total v1.ResourceList
}

// New returns an empty Cache.
func New() *Cache { return &Cache{infos: make(map[string]*info)} }

// AddPod records podKey's request against quotaKey. Idempotent per podKey; if
// podKey is already present with a different request, the total is adjusted.
func (c *Cache) AddPod(quotaKey, podKey string, req v1.ResourceList) {
	c.mu.Lock()
	defer c.mu.Unlock()
	in := c.infos[quotaKey]
	if in == nil {
		in = &info{pods: make(map[string]v1.ResourceList), total: v1.ResourceList{}}
		c.infos[quotaKey] = in
	}
	if prev, ok := in.pods[podKey]; ok {
		sub(in.total, prev)
	}
	req = req.DeepCopy()
	in.pods[podKey] = req
	add(in.total, req)
}

// RemovePod drops podKey's contribution from quotaKey. No-op if absent.
func (c *Cache) RemovePod(quotaKey, podKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	in := c.infos[quotaKey]
	if in == nil {
		return
	}
	if prev, ok := in.pods[podKey]; ok {
		sub(in.total, prev)
		delete(in.pods, podKey)
	}
	if len(in.pods) == 0 {
		delete(c.infos, quotaKey)
	}
}

// Used returns a copy of the summed request of all Pods accounted to quotaKey.
func (c *Cache) Used(quotaKey string) v1.ResourceList {
	c.mu.RLock()
	defer c.mu.RUnlock()
	in := c.infos[quotaKey]
	if in == nil {
		return v1.ResourceList{}
	}
	return in.total.DeepCopy()
}

func add(total, req v1.ResourceList) {
	for name, q := range req {
		cur := total[name]
		cur.Add(q)
		total[name] = cur
	}
}

func sub(total, req v1.ResourceList) {
	for name, q := range req {
		cur := total[name]
		cur.Sub(q)
		if cur.IsZero() {
			delete(total, name)
		} else {
			total[name] = cur
		}
	}
}
