// Package labels centralises the label and annotation keys mandated by
// the MLJob operator design (§6 Pod injection contract).
package labels

const (
	// JobIDLabel anchors the MLJob CR's stable UUID; mirrored on every
	// derived Pod so Compute can reverse-lookup orphans.
	JobIDLabel = "axisml.io/job-id"
	// QuotaLabel carries the bare quota name (e.g., "training") and is
	// distinct from KoordQuotaLabel (which holds the ElasticQuota CR
	// full name).
	QuotaLabel = "axisml.io/quota"
	// RoleLabel identifies the role within the job topology.
	RoleLabel = "axisml.io/role"

	// KoordQuotaLabel binds the Pod to a Koordinator ElasticQuota by
	// the quota CR's full name. Required for ElasticQuota plugin to
	// account the Pod's resources into status.used.
	KoordQuotaLabel = "quota.scheduling.koordinator.sh/name"

	// PodGroupLabel binds bare Pods to a scheduler-plugins PodGroup.
	PodGroupLabel = "pod-group.scheduling.sigs.k8s.io"

	// AppliedSpecAnnotation records a fingerprint of the immutable
	// portion of MLJob.spec (backend tuple + role topology) at first
	// observation. Dispatcher uses this to reject post-creation
	// mutations to backend.{name,engine} and roles[*].{name,replicas}.
	AppliedSpecAnnotation = "axisml.io/applied-spec"
)

// KoordSchedulerName is the name every AxisML workload Pod must use as
// spec.schedulerName.
const KoordSchedulerName = "koord-scheduler"
