package server

import "time"

// ClusterMeter is one resource dimension's utilisation on the cluster (or one
// pool): how much is in use against the schedulable total. Presentation
// (percentage, alert state) is derived client-side.
type ClusterMeter struct {
	Resource string  `json:"resource" desc:"Resource dimension (gpu, cpu, memory)."`
	Used     float64 `json:"used" desc:"Amount currently in use."`
	Total    float64 `json:"total" desc:"Schedulable total capacity."`
	Unit     string  `json:"unit,omitempty" desc:"Value unit (e.g. cards, cores, GiB)."`
}

// ClusterPoolUsage is the resource utilisation of one resource pool.
type ClusterPoolUsage struct {
	Pool   string         `json:"pool" desc:"Resource pool name."`
	Meters []ClusterMeter `json:"meters" desc:"Per-resource utilisation meters for the pool."`
}

// ClusterUsage is the dashboard's per-pool cluster resource-usage snapshot: one
// entry per resource pool the active tenant has quota in. There is no cross-pool
// aggregate — Platform folds cluster-manager per-(tenant, pool) usage into this
// list, one row per pool.
type ClusterUsage struct {
	Pools     []ClusterPoolUsage `json:"pools" desc:"Per-pool utilisation, one entry per pool the tenant has quota in."`
	Partial   bool               `json:"partial,omitempty" desc:"True when one or more pools could not be sampled and were omitted from the snapshot."`
	UpdatedAt time.Time          `json:"updatedAt" desc:"Time the snapshot was sampled."`
}

// ActivityItem is one entry in the dashboard's recent-activity feed.
type ActivityItem struct {
	ID        string    `json:"id" desc:"Stable activity entry identifier."`
	Kind      string    `json:"kind" desc:"Subject resource kind (workspace, job, experiment, run, mlservice, trafficpolicy)."`
	Name      string    `json:"name" desc:"Subject resource name."`
	Action    string    `json:"action" desc:"What happened (created, started, stopped, succeeded, failed, deleted, ...)."`
	Phase     string    `json:"phase,omitempty" desc:"Subject's phase at the time of the entry."`
	Actor     string    `json:"actor,omitempty" desc:"Username that triggered the activity."`
	Timestamp time.Time `json:"timestamp" desc:"Time the activity occurred."`
}

// ActivityList is a page of ActivityItem, newest first.
type ActivityList struct {
	Items []ActivityItem `json:"items" desc:"Activity entries, newest first."`
	Count int            `json:"count" binding:"min=0" desc:"Number of entries in this page."`
}
