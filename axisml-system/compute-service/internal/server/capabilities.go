package server

// Capabilities describes what Compute Service supports in the current deployment
// form. The Kubernetes form runs CRs through operators onto a axisml-scheduler-managed
// cluster (ElasticQuota-enforced); the standalone deployment runs them through the in-process
// Standalone runtime. Both forms use Compute's queue quota admission, while only
// Kubernetes has scheduler-side ElasticQuota enforcement. The composition root
// declares the form-specific values it assembled.
type Capabilities struct {
	// Runtime is the workload execution engine: "kubernetes" or "standalone".
	Runtime string `json:"runtime" desc:"Workload execution engine for this deployment form (kubernetes or standalone)."`
	// QuotaEnforcement reports whether the scheduler admits pods against an
	// ElasticQuota (true on Kubernetes, false on a standalone runtime). Queue
	// quota enforcement is reported independently below.
	QuotaEnforcement bool `json:"quotaEnforcement" desc:"True when Kubernetes scheduler-side ElasticQuota enforcement is enabled; false in standalone."`
	// RunQueueAdmission reports that MLRuns remain Queued until Compute has
	// reserved both runtime capacity and tenant quota.
	RunQueueAdmission bool `json:"runQueueAdmission" desc:"True when Compute keeps Runs queued until capacity and quota are available."`
	// RunPriority reports support for scheduling.axisml.io/priority.
	RunPriority bool `json:"runPriority" desc:"True when queued Runs are considered by priority then FIFO."`
	// RunQueueQuotaEnforcement is the cross-runtime queue quota contract. It is
	// distinct from legacy scheduler-side quotaEnforcement.
	RunQueueQuotaEnforcement bool `json:"runQueueQuotaEnforcement" desc:"True when Run queue admission enforces Tenant pool quotas in this deployment form."`
}
