package server

// Capabilities describes what Compute Service supports in the current deployment
// form. The Kubernetes form runs CRs through operators onto a axisml-scheduler-managed
// cluster (ElasticQuota-enforced); the standalone deployment runs them through the in-process
// Standalone runtime (no scheduler, no quota admission). The composition root
// declares the form-specific values it assembled.
type Capabilities struct {
	// Runtime is the workload execution engine: "kubernetes" or "standalone".
	Runtime string `json:"runtime" desc:"Workload execution engine for this deployment form (kubernetes or standalone)."`
	// QuotaEnforcement reports whether the scheduler admits pods against an
	// ElasticQuota (true on Kubernetes, false on a standalone runtime).
	QuotaEnforcement bool `json:"quotaEnforcement" desc:"True when the scheduler admits pods against an ElasticQuota; false when the runtime has no quota admission."`
}
