package server

import "time"

// ExperimentView is a Platform-owned reusable training-experiment template. It
// is the training-specialized form of a Job: its Spec is isomorphic to JobSpec
// (training hyperparameters are role args/env, not separately modelled). Each
// run produces a compute MLRun named <experiment>-<n> labelled
// axisml.io/experiment.
type ExperimentView struct {
	ID          UUID      `json:"id"`
	Namespace   string    `json:"namespace"`
	TenantName  string    `json:"tenantName"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName,omitempty"`
	Description string    `json:"description,omitempty"`
	Owner       string    `json:"owner"`
	OwnerID     UUID      `json:"ownerId,omitempty"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ExperimentList is a page of ExperimentView.
type ExperimentList struct {
	Items         []ExperimentView `json:"items"`
	Count         int              `json:"count" binding:"min=0"`
	ContinueToken string           `json:"continueToken,omitempty"`
	Partial       bool             `json:"partial,omitempty"`
}

// ExperimentCreateInput is the body of POST /experiments.
type ExperimentCreateInput struct {
	Name        string    `json:"name" binding:"required,dns1123,min=1,max=63"`
	DisplayName string    `json:"displayName,omitempty" binding:"max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec" binding:"required"`
}

// ExperimentPatchInput is the body of PATCH /experiments/{name}. Edits only
// affect Runs triggered afterwards.
type ExperimentPatchInput struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec,omitempty"`
}

// TensorBoard is an on-demand, read-only metric view for an experiment (or a
// selected set of its Runs), backed by a compute MLService(kind=tensorboard).
type TensorBoard struct {
	Name      string           `json:"name"`
	URL       string           `json:"url,omitempty"`
	Phase     TensorBoardPhase `json:"phase,omitempty"`
	Message   string           `json:"message,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
}

// TensorBoardRequest optionally scopes a TensorBoard to specific Runs; empty
// aggregates every Run of the experiment.
type TensorBoardRequest struct {
	Runs []string `json:"runs,omitempty"`
}
