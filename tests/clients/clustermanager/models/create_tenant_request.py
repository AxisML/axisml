from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.create_tenant_request_annotations import (
        CreateTenantRequestAnnotations,
    )
    from ..models.create_tenant_request_labels import CreateTenantRequestLabels
    from ..models.server_quota import ServerQuota
    from ..models.tenantv_1_alpha_1_init_resources import Tenantv1Alpha1InitResources
    from ..models.tenantv_1_alpha_1_namespace_spec import Tenantv1Alpha1NamespaceSpec


T = TypeVar("T", bound="CreateTenantRequest")


@_attrs_define
class CreateTenantRequest:
    """
    Example:
        {'labels': {'displayName': 'Vision Team'}, 'name': 'team-vision', 'namespace': {'name': 'team-vision'},
            'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}]}

    Attributes:
        annotations (CreateTenantRequestAnnotations | Unset): User-defined annotations to set on the tenant.
        init_resources (None | Tenantv1Alpha1InitResources | Unset): Per-tenant init resources to seed on provisioning.
        labels (CreateTenantRequestLabels | Unset): User-defined labels to set on the tenant.
        name (str | Unset): Tenant identifier to create; becomes the CR name, namespace, and partition string.
        namespace (None | Tenantv1Alpha1NamespaceSpec | Unset): Optional namespace specification; defaults are derived
            from the tenant name when omitted.
        quotas (list[ServerQuota] | Unset): Initial per-pool quotas to grant the tenant. Each item must use either units
            or quota.
    """

    annotations: CreateTenantRequestAnnotations | Unset = UNSET
    init_resources: None | Tenantv1Alpha1InitResources | Unset = UNSET
    labels: CreateTenantRequestLabels | Unset = UNSET
    name: str | Unset = UNSET
    namespace: None | Tenantv1Alpha1NamespaceSpec | Unset = UNSET
    quotas: list[ServerQuota] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
        )
        from ..models.tenantv_1_alpha_1_namespace_spec import (
            Tenantv1Alpha1NamespaceSpec,
        )

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        init_resources: dict[str, Any] | None | Unset
        if isinstance(self.init_resources, Unset):
            init_resources = UNSET
        elif isinstance(self.init_resources, Tenantv1Alpha1InitResources):
            init_resources = self.init_resources.to_dict()
        else:
            init_resources = self.init_resources

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        name = self.name

        namespace: dict[str, Any] | None | Unset
        if isinstance(self.namespace, Unset):
            namespace = UNSET
        elif isinstance(self.namespace, Tenantv1Alpha1NamespaceSpec):
            namespace = self.namespace.to_dict()
        else:
            namespace = self.namespace

        quotas: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.quotas, Unset):
            quotas = []
            for quotas_item_data in self.quotas:
                quotas_item = quotas_item_data.to_dict()
                quotas.append(quotas_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if init_resources is not UNSET:
            field_dict["initResources"] = init_resources
        if labels is not UNSET:
            field_dict["labels"] = labels
        if name is not UNSET:
            field_dict["name"] = name
        if namespace is not UNSET:
            field_dict["namespace"] = namespace
        if quotas is not UNSET:
            field_dict["quotas"] = quotas

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.create_tenant_request_annotations import (
            CreateTenantRequestAnnotations,
        )
        from ..models.create_tenant_request_labels import CreateTenantRequestLabels
        from ..models.server_quota import ServerQuota
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
        )
        from ..models.tenantv_1_alpha_1_namespace_spec import (
            Tenantv1Alpha1NamespaceSpec,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: CreateTenantRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = CreateTenantRequestAnnotations.from_dict(_annotations)

        def _parse_init_resources(
            data: object,
        ) -> None | Tenantv1Alpha1InitResources | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                init_resources_type_1 = Tenantv1Alpha1InitResources.from_dict(data)

                return init_resources_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Tenantv1Alpha1InitResources | Unset, data)

        init_resources = _parse_init_resources(d.pop("initResources", UNSET))

        _labels = d.pop("labels", UNSET)
        labels: CreateTenantRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = CreateTenantRequestLabels.from_dict(_labels)

        name = d.pop("name", UNSET)

        def _parse_namespace(
            data: object,
        ) -> None | Tenantv1Alpha1NamespaceSpec | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                namespace_type_1 = Tenantv1Alpha1NamespaceSpec.from_dict(data)

                return namespace_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Tenantv1Alpha1NamespaceSpec | Unset, data)

        namespace = _parse_namespace(d.pop("namespace", UNSET))

        _quotas = d.pop("quotas", UNSET)
        quotas: list[ServerQuota] | Unset = UNSET
        if _quotas is not UNSET:
            quotas = []
            for quotas_item_data in _quotas:
                quotas_item = ServerQuota.from_dict(quotas_item_data)

                quotas.append(quotas_item)

        create_tenant_request = cls(
            annotations=annotations,
            init_resources=init_resources,
            labels=labels,
            name=name,
            namespace=namespace,
            quotas=quotas,
        )

        create_tenant_request.additional_properties = d
        return create_tenant_request

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
