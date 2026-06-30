package server

import "time"

// Experiment is a Platform-owned reusable training-experiment template. It
// is the training-specialized form of a Job: its Spec is isomorphic to JobSpec
// (training hyperparameters are role args/env, not separately modelled). Each
// run produces a compute MLRun named <experiment>-<n> labelled
// compute.axisml.io/experiment.
type Experiment struct {
	ID          UUID      `json:"id" desc:"Stable experiment identifier."`
	Namespace   string    `json:"namespace" desc:"Platform tenant namespace the experiment belongs to."`
	TenantName  string    `json:"tenantName" desc:"Tenant identifier owning the experiment."`
	Name        string    `json:"name" desc:"Experiment definition name (unique within the tenant)."`
	DisplayName string    `json:"displayName,omitempty" desc:"Human-readable experiment label."`
	Description string    `json:"description,omitempty" desc:"Free-text experiment description."`
	Owner       string    `json:"owner" desc:"Username of the experiment owner."`
	OwnerID     UUID      `json:"ownerId,omitempty" desc:"User ID of the experiment owner."`
	Labels      StringMap `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        JobSpec   `json:"spec" desc:"Reusable run template (isomorphic to a Job spec)."`
	CreatedAt   time.Time `json:"createdAt" desc:"Time the experiment was created."`
	UpdatedAt   time.Time `json:"updatedAt" desc:"Time the experiment was last updated."`
}

// ExperimentList is a page of Experiment.
type ExperimentList struct {
	Items         []Experiment `json:"items" desc:"Experiments in this page."`
	Count         int          `json:"count" binding:"min=0" desc:"Number of experiments in this page."`
	ContinueToken string       `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool         `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// ExperimentCreateRequest is the body of POST /experiments.
type ExperimentCreateRequest struct {
	Name        string    `json:"name" binding:"required,dns1123,min=1,max=63" desc:"Experiment definition name (unique within the tenant)."`
	DisplayName string    `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable experiment label."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Free-text experiment description."`
	Labels      StringMap `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        JobSpec   `json:"spec" binding:"required" desc:"Reusable run template."`
}

// ExperimentPatchRequest is the body of PATCH /experiments/{name}. Edits only
// affect Runs triggered afterwards.
type ExperimentPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable experiment label."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text experiment description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap `json:"annotations,omitempty" desc:"Replacement annotation set."`
	Spec        JobSpec   `json:"spec,omitempty" desc:"Replacement run template (affects only Runs triggered afterwards)."`
}

// TensorBoard is an on-demand, read-only metric view for an experiment (or a
// selected set of its Runs), backed by a compute MLService(kind=tensorboard).
type TensorBoard struct {
	Name      string           `json:"name" desc:"TensorBoard instance name."`
	URL       string           `json:"url,omitempty" desc:"Endpoint URL for the TensorBoard UI once running."`
	Phase     TensorBoardPhase `json:"phase,omitempty" desc:"Current TensorBoard lifecycle phase."`
	Message   string           `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	CreatedAt time.Time        `json:"createdAt" desc:"Time the TensorBoard was created."`
}

// TensorBoardRequest optionally scopes a TensorBoard to specific Runs; empty
// aggregates every Run of the experiment.
type TensorBoardRequest struct {
	Runs []string `json:"runs,omitempty" desc:"Run names to scope the TensorBoard to; empty aggregates every Run of the experiment."`
}
