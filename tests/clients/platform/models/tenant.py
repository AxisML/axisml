from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.tenant_phase import TenantPhase
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.init_resources import InitResources
    from ..models.quota import Quota
    from ..models.string_map import StringMap
    from ..models.tenant_status import TenantStatus
    from ..models.tenant_volume import TenantVolume


T = TypeVar("T", bound="Tenant")


@_attrs_define
class Tenant:
    """
    Example:
        {'activeExperimentRuns': 2, 'activeJobRuns': 3, 'annotations': {'axisml.io/created-by': 'system-admin'},
            'createdAt': '2026-06-20T08:00:00Z', 'description': 'Computer-vision model training and inference team.',
            'displayName': 'Vision Team', 'identifier': 'team-vision', 'initResources': {'configMaps': [{'name': 'shared-
            config', 'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-system'}}], 'imagePullSecrets':
            [{'name': 'registry-pull', 'sourceSecretRef': {'name': 'registry-pull', 'namespace': 'axisml-system'}}],
            'secrets': [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-key', 'namespace': 'axisml-
            system'}, 'type': 'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-pull'], 'name': 'trainer',
            'rbac': {'roleRefs': [{'kind': 'ClusterRole', 'name': 'edit'}]}}]}, 'kubernetesNamespace': 'axisml-team-vision',
            'labels': {'team': 'vision'}, 'memberCount': 8, 'onlineServices': 1, 'owner': 'li.wei', 'phase': 'Active',
            'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}], 'status': {'conditions':
            [{'lastTransitionTime': '2026-06-28T09:30:00Z', 'message': 'Namespace and quota provisioned.', 'reason':
            'Provisioned', 'status': 'True', 'type': 'Ready'}], 'message': 'Tenant is active.', 'quotas': [{'pool':
            'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x', 'used': 3}]}]}, 'suspended': False, 'updatedAt':
            '2026-06-28T09:30:00Z'}

    Attributes:
        active_experiment_runs (int): Number of active experiment runs in the tenant.
        active_job_runs (int): Number of active job runs in the tenant.
        created_at (datetime.datetime): Time the tenant was created.
        display_name (str): Human-readable tenant name.
        identifier (str): Stable logical tenant scope used across Platform, compute and artifacts.
        kubernetes_namespace (str): Physical Kubernetes namespace backing the tenant (may be shared).
        member_count (int): Number of members bound to the tenant.
        online_services (int): Number of online ML services in the tenant.
        phase (TenantPhase):
        suspended (bool): Whether new workloads are blocked (Platform-enforced).
        updated_at (datetime.datetime): Time the tenant was last updated.
        annotations (StringMap | Unset):
        description (str | Unset): Free-text tenant description.
        init_resources (InitResources | Unset):  Example: {'configMaps': [{'name': 'shared-config',
            'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-system'}}], 'imagePullSecrets': [{'name':
            'registry-pull', 'sourceSecretRef': {'name': 'registry-pull', 'namespace': 'axisml-system'}}], 'secrets':
            [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-key', 'namespace': 'axisml-system'}, 'type':
            'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-pull'], 'name': 'trainer', 'rbac': {'roleRefs':
            [{'kind': 'ClusterRole', 'name': 'edit'}]}}]}.
        labels (StringMap | Unset):
        owner (str | Unset): Username of the tenant owner.
        quotas (list[Quota] | Unset): Per-pool resource quota allocations.
        status (TenantStatus | Unset):  Example: {'conditions': [{'lastTransitionTime': '2026-06-28T09:30:00Z',
            'message': 'Namespace and quota provisioned.', 'reason': 'Provisioned', 'status': 'True', 'type': 'Ready'}],
            'message': 'Tenant is active.', 'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x',
            'used': 3}]}]}.
        volumes (list[TenantVolume] | Unset): Predefined data volumes ensured to exist in the tenant namespace.
    """

    active_experiment_runs: int
    active_job_runs: int
    created_at: datetime.datetime
    display_name: str
    identifier: str
    kubernetes_namespace: str
    member_count: int
    online_services: int
    phase: TenantPhase
    suspended: bool
    updated_at: datetime.datetime
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    init_resources: InitResources | Unset = UNSET
    labels: StringMap | Unset = UNSET
    owner: str | Unset = UNSET
    quotas: list[Quota] | Unset = UNSET
    status: TenantStatus | Unset = UNSET
    volumes: list[TenantVolume] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        active_experiment_runs = self.active_experiment_runs

        active_job_runs = self.active_job_runs

        created_at = self.created_at.isoformat()

        display_name = self.display_name

        identifier = self.identifier

        kubernetes_namespace = self.kubernetes_namespace

        member_count = self.member_count

        online_services = self.online_services

        phase = self.phase.value

        suspended = self.suspended

        updated_at = self.updated_at.isoformat()

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        init_resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.init_resources, Unset):
            init_resources = self.init_resources.to_dict()

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        owner = self.owner

        quotas: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.quotas, Unset):
            quotas = []
            for quotas_item_data in self.quotas:
                quotas_item = quotas_item_data.to_dict()
                quotas.append(quotas_item)

        status: dict[str, Any] | Unset = UNSET
        if not isinstance(self.status, Unset):
            status = self.status.to_dict()

        volumes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volumes, Unset):
            volumes = []
            for volumes_item_data in self.volumes:
                volumes_item = volumes_item_data.to_dict()
                volumes.append(volumes_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "activeExperimentRuns": active_experiment_runs,
                "activeJobRuns": active_job_runs,
                "createdAt": created_at,
                "displayName": display_name,
                "identifier": identifier,
                "kubernetesNamespace": kubernetes_namespace,
                "memberCount": member_count,
                "onlineServices": online_services,
                "phase": phase,
                "suspended": suspended,
                "updatedAt": updated_at,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if init_resources is not UNSET:
            field_dict["initResources"] = init_resources
        if labels is not UNSET:
            field_dict["labels"] = labels
        if owner is not UNSET:
            field_dict["owner"] = owner
        if quotas is not UNSET:
            field_dict["quotas"] = quotas
        if status is not UNSET:
            field_dict["status"] = status
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.init_resources import InitResources
        from ..models.quota import Quota
        from ..models.string_map import StringMap
        from ..models.tenant_status import TenantStatus
        from ..models.tenant_volume import TenantVolume

        d = dict(src_dict)
        active_experiment_runs = d.pop("activeExperimentRuns")

        active_job_runs = d.pop("activeJobRuns")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        display_name = d.pop("displayName")

        identifier = d.pop("identifier")

        kubernetes_namespace = d.pop("kubernetesNamespace")

        member_count = d.pop("memberCount")

        online_services = d.pop("onlineServices")

        phase = TenantPhase(d.pop("phase"))

        suspended = d.pop("suspended")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _init_resources = d.pop("initResources", UNSET)
        init_resources: InitResources | Unset
        if isinstance(_init_resources, Unset):
            init_resources = UNSET
        else:
            init_resources = InitResources.from_dict(_init_resources)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        owner = d.pop("owner", UNSET)

        _quotas = d.pop("quotas", UNSET)
        quotas: list[Quota] | Unset = UNSET
        if _quotas is not UNSET:
            quotas = []
            for quotas_item_data in _quotas:
                quotas_item = Quota.from_dict(quotas_item_data)

                quotas.append(quotas_item)

        _status = d.pop("status", UNSET)
        status: TenantStatus | Unset
        if isinstance(_status, Unset):
            status = UNSET
        else:
            status = TenantStatus.from_dict(_status)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[TenantVolume] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = TenantVolume.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        tenant = cls(
            active_experiment_runs=active_experiment_runs,
            active_job_runs=active_job_runs,
            created_at=created_at,
            display_name=display_name,
            identifier=identifier,
            kubernetes_namespace=kubernetes_namespace,
            member_count=member_count,
            online_services=online_services,
            phase=phase,
            suspended=suspended,
            updated_at=updated_at,
            annotations=annotations,
            description=description,
            init_resources=init_resources,
            labels=labels,
            owner=owner,
            quotas=quotas,
            status=status,
            volumes=volumes,
        )

        tenant.additional_properties = d
        return tenant

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
