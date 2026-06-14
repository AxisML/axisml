//go:build e2e

package e2e

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

	mljobv1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// These are e2e-local mirrors of the services' HTTP DTOs. The real DTOs live
// under each component's internal/ package and cannot be imported across
// modules, so we re-declare the JSON contract here (kept in sync by the
// doc-gen'd OpenAPI specs). Where a DTO embeds a public CR type we reuse it.

// ---------- cluster-manager ----------

type cmCreateUnitReq struct {
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Requests     corev1.ResourceList `json:"requests"`
	Limits       corev1.ResourceList `json:"limits"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

type cmCreatePoolReq struct {
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Units        []cmCreateUnitReq   `json:"units,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

type cmUnitDTO struct {
	Name     string              `json:"name"`
	Requests corev1.ResourceList `json:"requests"`
	Limits   corev1.ResourceList `json:"limits"`
}

type cmPoolDTO struct {
	Name  string      `json:"name"`
	Units []cmUnitDTO `json:"units"`
}

// ---------- compute-service: tenant ----------

type csNamespaceSpec struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type csQuotaSpec struct {
	Pool string            `json:"pool"`
	Name string            `json:"name"`
	Min  map[string]string `json:"min,omitempty"`
	Max  map[string]string `json:"max"`
}

type csCreateTenantReq struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName,omitempty"`
	Description string          `json:"description,omitempty"`
	Namespace   csNamespaceSpec `json:"namespace"`
	Quotas      []csQuotaSpec   `json:"quotas,omitempty"`
}

type csTenantResp struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Namespace csNamespaceSpec `json:"namespace"`
	Quotas    []csQuotaSpec   `json:"quotas,omitempty"`
	Phase     string          `json:"phase"`
}

// ---------- compute-service: job ----------

type csCreateJobReq struct {
	Name     string               `json:"name"`
	PoolName string               `json:"poolName"`
	UnitName string               `json:"unitName"`
	Quota    string               `json:"quota"`
	Backend  *mljobv1.BackendSpec `json:"backend,omitempty"`
	Roles    []mljobv1.RoleSpec   `json:"roles"`
}

type csJobView struct {
	ID        string            `json:"id"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Phase     string            `json:"phase"`
	Spec      mljobv1.MLJobSpec `json:"spec"`
	Status    json.RawMessage   `json:"status"`
}

// ---------- compute-service: service ----------

type csWorkspaceStorage struct {
	Size         string `json:"size"`
	StorageClass string `json:"storageClass,omitempty"`
}

type csCreateServiceReq struct {
	Name             string                 `json:"name"`
	Kind             string                 `json:"kind,omitempty"`
	PoolName         string                 `json:"poolName"`
	UnitName         string                 `json:"unitName"`
	Quota            string                 `json:"quota"`
	Backend          *mlservicev1.Backend   `json:"backend,omitempty"`
	Roles            []mlservicev1.RoleSpec `json:"roles"`
	Route            *mlservicev1.Route     `json:"route,omitempty"`
	WorkspaceStorage *csWorkspaceStorage    `json:"workspaceStorage,omitempty"`
}

type csScaleReq struct {
	Replicas int32 `json:"replicas"`
}

type csServiceView struct {
	ID        string                    `json:"id"`
	Namespace string                    `json:"namespace"`
	Name      string                    `json:"name"`
	Kind      string                    `json:"kind"`
	Phase     string                    `json:"phase"`
	Spec      mlservicev1.MLServiceSpec `json:"spec"`
	Status    json.RawMessage           `json:"status"`
}

// ---------- compute-service: traffic policy ----------

type csCreateTrafficPolicyReq struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName,omitempty"`
	Mode        string                 `json:"mode"`
	Endpoint    mltpv1.Endpoint        `json:"endpoint"`
	Backends    []mltpv1.BackendMember `json:"backends"`
}

type csTrafficWeightUpdate struct {
	ServiceName string `json:"serviceName"`
	Weight      int32  `json:"weight"`
}

type csTrafficSplitReq struct {
	Backends []csTrafficWeightUpdate `json:"backends"`
}

type csTrafficPolicyView struct {
	ID        string                     `json:"id"`
	Namespace string                     `json:"namespace"`
	Name      string                     `json:"name"`
	Mode      string                     `json:"mode"`
	Phase     string                     `json:"phase"`
	Spec      mltpv1.MLTrafficPolicySpec `json:"spec"`
	Status    json.RawMessage            `json:"status"`
}

// ---------- artifact-hub ----------

type ahInitiateReq struct {
	Version     string            `json:"version"`
	Spec        map[string]any    `json:"spec"`
	Visibility  string            `json:"visibility,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ahView struct {
	ID          string            `json:"id"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Visibility  string            `json:"visibility"`
	Status      string            `json:"status"`
	Digest      string            `json:"digest,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Spec        map[string]any    `json:"spec"`
}

type ahUploadCreds struct {
	StorageKind string          `json:"storageKind"`
	URI         string          `json:"uri"`
	Credentials json.RawMessage `json:"credentials"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}

type ahInitiateResult struct {
	Artifact ahView        `json:"artifact"`
	Upload   ahUploadCreds `json:"upload"`
}

type ahCompleteReq struct {
	Digest string `json:"digest"`
}

type ahResolveResult struct {
	StorageKind     string          `json:"storageKind"`
	URI             string          `json:"uri"`
	Digest          string          `json:"digest,omitempty"`
	Visibility      string          `json:"visibility,omitempty"`
	PullCredentials json.RawMessage `json:"pullCredentials,omitempty"`
	ExpiresAt       *time.Time      `json:"expiresAt,omitempty"`
}

type ahPatchReq struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
