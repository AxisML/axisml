from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.init_resources import InitResources
    from ..models.quota import Quota
    from ..models.string_map import StringMap
    from ..models.tenant_volume import TenantVolume


T = TypeVar("T", bound="TenantCreateRequest")


@_attrs_define
class TenantCreateRequest:
    """
    Example:
        {'description': 'Computer-vision model training and inference team.', 'displayName': 'Vision Team',
            'identifier': 'team-vision', 'initialAdmin': 'li.wei@example.com', 'kubernetesNamespace': 'axisml-team-vision',
            'labels': {'team': 'vision'}, 'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName':
            'a100-2x'}]}]}

    Attributes:
        display_name (str): Human-readable tenant name.
        identifier (str): Stable logical tenant scope (becomes the Tenant CR name).
        initial_admin (str): Email or username of the first tenant-admin member.
        kubernetes_namespace (str): Physical Kubernetes namespace to back the tenant (may be shared).
        annotations (StringMap | Unset):
        description (str | Unset): Free-text tenant description.
        init_resources (InitResources | Unset):  Example: {'configMaps': [{'name': 'shared-config',
            'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-system'}}], 'imagePullSecrets': [{'name':
            'registry-pull', 'sourceSecretRef': {'name': 'registry-pull', 'namespace': 'axisml-system'}}], 'secrets':
            [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-key', 'namespace': 'axisml-system'}, 'type':
            'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-pull'], 'name': 'trainer', 'rbac': {'roleRefs':
            [{'kind': 'ClusterRole', 'name': 'edit'}]}}]}.
        labels (StringMap | Unset):
        quotas (list[Quota] | Unset): Initial per-pool resource quota allocations.
        volumes (list[TenantVolume] | Unset): Predefined data volumes to ensure in the tenant namespace at provisioning.
    """

    display_name: str
    identifier: str
    initial_admin: str
    kubernetes_namespace: str
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    init_resources: InitResources | Unset = UNSET
    labels: StringMap | Unset = UNSET
    quotas: list[Quota] | Unset = UNSET
    volumes: list[TenantVolume] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        display_name = self.display_name

        identifier = self.identifier

        initial_admin = self.initial_admin

        kubernetes_namespace = self.kubernetes_namespace

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

        quotas: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.quotas, Unset):
            quotas = []
            for quotas_item_data in self.quotas:
                quotas_item = quotas_item_data.to_dict()
                quotas.append(quotas_item)

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
                "displayName": display_name,
                "identifier": identifier,
                "initialAdmin": initial_admin,
                "kubernetesNamespace": kubernetes_namespace,
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
        if quotas is not UNSET:
            field_dict["quotas"] = quotas
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.init_resources import InitResources
        from ..models.quota import Quota
        from ..models.string_map import StringMap
        from ..models.tenant_volume import TenantVolume

        d = dict(src_dict)
        display_name = d.pop("displayName")

        identifier = d.pop("identifier")

        initial_admin = d.pop("initialAdmin")

        kubernetes_namespace = d.pop("kubernetesNamespace")

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

        _quotas = d.pop("quotas", UNSET)
        quotas: list[Quota] | Unset = UNSET
        if _quotas is not UNSET:
            quotas = []
            for quotas_item_data in _quotas:
                quotas_item = Quota.from_dict(quotas_item_data)

                quotas.append(quotas_item)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[TenantVolume] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = TenantVolume.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        tenant_create_request = cls(
            display_name=display_name,
            identifier=identifier,
            initial_admin=initial_admin,
            kubernetes_namespace=kubernetes_namespace,
            annotations=annotations,
            description=description,
            init_resources=init_resources,
            labels=labels,
            quotas=quotas,
            volumes=volumes,
        )

        tenant_create_request.additional_properties = d
        return tenant_create_request

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
