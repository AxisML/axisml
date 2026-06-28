package server

// Capabilities describes what Compute Service supports in the current deployment
// form. The Kubernetes form runs CRs through operators onto a koord-scheduled
// cluster (ElasticQuota-enforced); the Lite form runs them through the in-process
// Standalone runtime (no scheduler, no quota admission). The composition root
// declares the form-specific values it assembled.
type Capabilities struct {
	// Runtime is the workload execution engine: "kubernetes" or "standalone".
	Runtime string `json:"runtime" desc:"Workload execution engine for this deployment form (kubernetes or standalone)."`
	// QuotaEnforcement reports whether the scheduler admits pods against an
	// ElasticQuota (true on Kubernetes, false on the Lite Standalone runtime).
	QuotaEnforcement bool `json:"quotaEnforcement" desc:"True when the scheduler admits pods against an ElasticQuota (Kubernetes form); false on the Lite Standalone runtime."`
}
