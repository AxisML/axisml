"""Contains all the data models used in inputs/outputs"""

from .capabilities import Capabilities
from .cluster_manager_error import ClusterManagerError
from .corev_1_toleration import Corev1Toleration
from .create_resource_pool_request import CreateResourcePoolRequest
from .create_resource_pool_request_annotations import (
    CreateResourcePoolRequestAnnotations,
)
from .create_resource_pool_request_labels import CreateResourcePoolRequestLabels
from .create_resource_pool_request_node_selector import (
    CreateResourcePoolRequestNodeSelector,
)
from .create_resource_unit_request import CreateResourceUnitRequest
from .create_resource_unit_request_annotations import (
    CreateResourceUnitRequestAnnotations,
)
from .create_resource_unit_request_limits import CreateResourceUnitRequestLimits
from .create_resource_unit_request_node_selector import (
    CreateResourceUnitRequestNodeSelector,
)
from .create_resource_unit_request_requests import CreateResourceUnitRequestRequests
from .create_tenant_request import CreateTenantRequest
from .create_tenant_request_annotations import CreateTenantRequestAnnotations
from .create_tenant_request_labels import CreateTenantRequestLabels
from .create_volume_request import CreateVolumeRequest
from .create_volume_request_labels import CreateVolumeRequestLabels
from .patch_quota_request import PatchQuotaRequest
from .patch_resource_pool_request import PatchResourcePoolRequest
from .patch_resource_pool_request_annotations import PatchResourcePoolRequestAnnotations
from .patch_resource_pool_request_labels import PatchResourcePoolRequestLabels
from .patch_resource_pool_request_node_selector import (
    PatchResourcePoolRequestNodeSelector,
)
from .patch_resource_unit_request import PatchResourceUnitRequest
from .patch_resource_unit_request_annotations import PatchResourceUnitRequestAnnotations
from .patch_resource_unit_request_limits import PatchResourceUnitRequestLimits
from .patch_resource_unit_request_node_selector import (
    PatchResourceUnitRequestNodeSelector,
)
from .patch_resource_unit_request_requests import PatchResourceUnitRequestRequests
from .patch_tenant_request import PatchTenantRequest
from .patch_tenant_request_annotations import PatchTenantRequestAnnotations
from .patch_tenant_request_labels import PatchTenantRequestLabels
from .patch_tenant_request_namespace_annotations import (
    PatchTenantRequestNamespaceAnnotations,
)
from .patch_tenant_request_namespace_labels import PatchTenantRequestNamespaceLabels
from .patch_volume_request import PatchVolumeRequest
from .patch_volume_request_labels import PatchVolumeRequestLabels
from .pool_metric_point import PoolMetricPoint
from .pool_metric_series import PoolMetricSeries
from .pool_usage import PoolUsage
from .quota import Quota
from .quota_list import QuotaList
from .rbacv_1_policy_rule import Rbacv1PolicyRule
from .resource_meter import ResourceMeter
from .resource_pool import ResourcePool
from .resource_pool_annotations import ResourcePoolAnnotations
from .resource_pool_labels import ResourcePoolLabels
from .resource_pool_list import ResourcePoolList
from .resource_pool_node_selector import ResourcePoolNodeSelector
from .resource_unit import ResourceUnit
from .resource_unit_annotations import ResourceUnitAnnotations
from .resource_unit_limits import ResourceUnitLimits
from .resource_unit_list import ResourceUnitList
from .resource_unit_node_selector import ResourceUnitNodeSelector
from .resource_unit_requests import ResourceUnitRequests
from .server_create_resource_unit_request import ServerCreateResourceUnitRequest
from .server_create_resource_unit_request_annotations import (
    ServerCreateResourceUnitRequestAnnotations,
)
from .server_create_resource_unit_request_limits import (
    ServerCreateResourceUnitRequestLimits,
)
from .server_create_resource_unit_request_node_selector import (
    ServerCreateResourceUnitRequestNodeSelector,
)
from .server_create_resource_unit_request_requests import (
    ServerCreateResourceUnitRequestRequests,
)
from .server_pool_metric_point import ServerPoolMetricPoint
from .server_quota import ServerQuota
from .server_quota_resources import ServerQuotaResources
from .server_quota_resources_max import ServerQuotaResourcesMax
from .server_quota_resources_min import ServerQuotaResourcesMin
from .server_quota_status import ServerQuotaStatus
from .server_quota_status_used import ServerQuotaStatusUsed
from .server_quota_unit import ServerQuotaUnit
from .server_resource_meter import ServerResourceMeter
from .server_resource_pool import ServerResourcePool
from .server_resource_pool_annotations import ServerResourcePoolAnnotations
from .server_resource_pool_labels import ServerResourcePoolLabels
from .server_resource_pool_node_selector import ServerResourcePoolNodeSelector
from .server_resource_unit import ServerResourceUnit
from .server_resource_unit_annotations import ServerResourceUnitAnnotations
from .server_resource_unit_limits import ServerResourceUnitLimits
from .server_resource_unit_node_selector import ServerResourceUnitNodeSelector
from .server_resource_unit_requests import ServerResourceUnitRequests
from .server_storage_class import ServerStorageClass
from .server_tenant import ServerTenant
from .server_tenant_annotations import ServerTenantAnnotations
from .server_tenant_labels import ServerTenantLabels
from .server_tenant_status import ServerTenantStatus
from .server_volume import ServerVolume
from .server_volume_labels import ServerVolumeLabels
from .server_volume_mount import ServerVolumeMount
from .server_volume_status import ServerVolumeStatus
from .set_quota_request import SetQuotaRequest
from .storage_class import StorageClass
from .storage_class_list import StorageClassList
from .tenant import Tenant
from .tenant_annotations import TenantAnnotations
from .tenant_labels import TenantLabels
from .tenant_list import TenantList
from .tenantv_1_alpha_1_config_map_spec import Tenantv1Alpha1ConfigMapSpec
from .tenantv_1_alpha_1_image_pull_secret_spec import Tenantv1Alpha1ImagePullSecretSpec
from .tenantv_1_alpha_1_init_resources import Tenantv1Alpha1InitResources
from .tenantv_1_alpha_1_namespace_spec import Tenantv1Alpha1NamespaceSpec
from .tenantv_1_alpha_1_namespace_spec_annotations import (
    Tenantv1Alpha1NamespaceSpecAnnotations,
)
from .tenantv_1_alpha_1_namespace_spec_labels import Tenantv1Alpha1NamespaceSpecLabels
from .tenantv_1_alpha_1_secret_spec import Tenantv1Alpha1SecretSpec
from .tenantv_1_alpha_1_service_account_spec import Tenantv1Alpha1ServiceAccountSpec
from .tenantv_1_alpha_1_source_config_map_ref import Tenantv1Alpha1SourceConfigMapRef
from .tenantv_1_alpha_1_source_secret_ref import Tenantv1Alpha1SourceSecretRef
from .tenantv_1_alpha_1_volume_spec import Tenantv1Alpha1VolumeSpec
from .tenantv_1_alpha_1rbac_role_ref import Tenantv1Alpha1RBACRoleRef
from .tenantv_1_alpha_1rbac_spec import Tenantv1Alpha1RBACSpec
from .volume import Volume
from .volume_labels import VolumeLabels
from .volume_list import VolumeList

__all__ = (
    "Capabilities",
    "ClusterManagerError",
    "Corev1Toleration",
    "CreateResourcePoolRequest",
    "CreateResourcePoolRequestAnnotations",
    "CreateResourcePoolRequestLabels",
    "CreateResourcePoolRequestNodeSelector",
    "CreateResourceUnitRequest",
    "CreateResourceUnitRequestAnnotations",
    "CreateResourceUnitRequestLimits",
    "CreateResourceUnitRequestNodeSelector",
    "CreateResourceUnitRequestRequests",
    "CreateTenantRequest",
    "CreateTenantRequestAnnotations",
    "CreateTenantRequestLabels",
    "CreateVolumeRequest",
    "CreateVolumeRequestLabels",
    "PatchQuotaRequest",
    "PatchResourcePoolRequest",
    "PatchResourcePoolRequestAnnotations",
    "PatchResourcePoolRequestLabels",
    "PatchResourcePoolRequestNodeSelector",
    "PatchResourceUnitRequest",
    "PatchResourceUnitRequestAnnotations",
    "PatchResourceUnitRequestLimits",
    "PatchResourceUnitRequestNodeSelector",
    "PatchResourceUnitRequestRequests",
    "PatchTenantRequest",
    "PatchTenantRequestAnnotations",
    "PatchTenantRequestLabels",
    "PatchTenantRequestNamespaceAnnotations",
    "PatchTenantRequestNamespaceLabels",
    "PatchVolumeRequest",
    "PatchVolumeRequestLabels",
    "PoolMetricPoint",
    "PoolMetricSeries",
    "PoolUsage",
    "Quota",
    "QuotaList",
    "Rbacv1PolicyRule",
    "ResourceMeter",
    "ResourcePool",
    "ResourcePoolAnnotations",
    "ResourcePoolLabels",
    "ResourcePoolList",
    "ResourcePoolNodeSelector",
    "ResourceUnit",
    "ResourceUnitAnnotations",
    "ResourceUnitLimits",
    "ResourceUnitList",
    "ResourceUnitNodeSelector",
    "ResourceUnitRequests",
    "ServerCreateResourceUnitRequest",
    "ServerCreateResourceUnitRequestAnnotations",
    "ServerCreateResourceUnitRequestLimits",
    "ServerCreateResourceUnitRequestNodeSelector",
    "ServerCreateResourceUnitRequestRequests",
    "ServerPoolMetricPoint",
    "ServerQuota",
    "ServerQuotaResources",
    "ServerQuotaResourcesMax",
    "ServerQuotaResourcesMin",
    "ServerQuotaStatus",
    "ServerQuotaStatusUsed",
    "ServerQuotaUnit",
    "ServerResourceMeter",
    "ServerResourcePool",
    "ServerResourcePoolAnnotations",
    "ServerResourcePoolLabels",
    "ServerResourcePoolNodeSelector",
    "ServerResourceUnit",
    "ServerResourceUnitAnnotations",
    "ServerResourceUnitLimits",
    "ServerResourceUnitNodeSelector",
    "ServerResourceUnitRequests",
    "ServerStorageClass",
    "ServerTenant",
    "ServerTenantAnnotations",
    "ServerTenantLabels",
    "ServerTenantStatus",
    "ServerVolume",
    "ServerVolumeLabels",
    "ServerVolumeMount",
    "ServerVolumeStatus",
    "SetQuotaRequest",
    "StorageClass",
    "StorageClassList",
    "Tenant",
    "TenantAnnotations",
    "TenantLabels",
    "TenantList",
    "Tenantv1Alpha1ConfigMapSpec",
    "Tenantv1Alpha1ImagePullSecretSpec",
    "Tenantv1Alpha1InitResources",
    "Tenantv1Alpha1NamespaceSpec",
    "Tenantv1Alpha1NamespaceSpecAnnotations",
    "Tenantv1Alpha1NamespaceSpecLabels",
    "Tenantv1Alpha1RBACRoleRef",
    "Tenantv1Alpha1RBACSpec",
    "Tenantv1Alpha1SecretSpec",
    "Tenantv1Alpha1ServiceAccountSpec",
    "Tenantv1Alpha1SourceConfigMapRef",
    "Tenantv1Alpha1SourceSecretRef",
    "Tenantv1Alpha1VolumeSpec",
    "Volume",
    "VolumeLabels",
    "VolumeList",
)
