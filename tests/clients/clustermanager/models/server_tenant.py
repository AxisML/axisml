from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_quota import ServerQuota
    from ..models.server_tenant_annotations import ServerTenantAnnotations
    from ..models.server_tenant_labels import ServerTenantLabels
    from ..models.server_tenant_status import ServerTenantStatus
    from ..models.tenantv_1_alpha_1_init_resources import Tenantv1Alpha1InitResources
    from ..models.tenantv_1_alpha_1_namespace_spec import Tenantv1Alpha1NamespaceSpec


T = TypeVar("T", bound="ServerTenant")


@_attrs_define
class ServerTenant:
    """
    Attributes:
        created_at (datetime.datetime): Tenant creation timestamp (RFC3339).
        name (str): Canonical tenant identifier; also the CR name, K8s namespace, and partition string.
        namespace (Tenantv1Alpha1NamespaceSpec):
        quotas (list[ServerQuota]): Per-pool quotas. Each item is returned either as units (business form) or quota
            (direct min/max form).
        annotations (ServerTenantAnnotations | Unset): User-defined annotations on the tenant.
        init_resources (None | Tenantv1Alpha1InitResources | Unset): Per-tenant init resources (Secrets, ConfigMaps,
            ServiceAccount, RBAC) seeded on provisioning.
        labels (ServerTenantLabels | Unset): User-defined labels on the tenant.
        phase (str | Unset): High-level provisioning phase of the tenant.
        resource_version (str | Unset): Opaque CR resourceVersion for optimistic concurrency.
        status (None | ServerTenantStatus | Unset): Live operator-written status read from the CR.
    """

    created_at: datetime.datetime
    name: str
    namespace: Tenantv1Alpha1NamespaceSpec
    quotas: list[ServerQuota]
    annotations: ServerTenantAnnotations | Unset = UNSET
    init_resources: None | Tenantv1Alpha1InitResources | Unset = UNSET
    labels: ServerTenantLabels | Unset = UNSET
    phase: str | Unset = UNSET
    resource_version: str | Unset = UNSET
    status: None | ServerTenantStatus | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.server_tenant_status import ServerTenantStatus
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
        )

        created_at = self.created_at.isoformat()

        name = self.name

        namespace = self.namespace.to_dict()

        quotas = []
        for quotas_item_data in self.quotas:
            quotas_item = quotas_item_data.to_dict()
            quotas.append(quotas_item)

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

        phase = self.phase

        resource_version = self.resource_version

        status: dict[str, Any] | None | Unset
        if isinstance(self.status, Unset):
            status = UNSET
        elif isinstance(self.status, ServerTenantStatus):
            status = self.status.to_dict()
        else:
            status = self.status

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "name": name,
                "namespace": namespace,
                "quotas": quotas,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if init_resources is not UNSET:
            field_dict["initResources"] = init_resources
        if labels is not UNSET:
            field_dict["labels"] = labels
        if phase is not UNSET:
            field_dict["phase"] = phase
        if resource_version is not UNSET:
            field_dict["resourceVersion"] = resource_version
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota import ServerQuota
        from ..models.server_tenant_annotations import ServerTenantAnnotations
        from ..models.server_tenant_labels import ServerTenantLabels
        from ..models.server_tenant_status import ServerTenantStatus
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
        )
        from ..models.tenantv_1_alpha_1_namespace_spec import (
            Tenantv1Alpha1NamespaceSpec,
        )

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        name = d.pop("name")

        namespace = Tenantv1Alpha1NamespaceSpec.from_dict(d.pop("namespace"))

        quotas = []
        _quotas = d.pop("quotas")
        for quotas_item_data in _quotas:
            quotas_item = ServerQuota.from_dict(quotas_item_data)

            quotas.append(quotas_item)

        _annotations = d.pop("annotations", UNSET)
        annotations: ServerTenantAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ServerTenantAnnotations.from_dict(_annotations)

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
        labels: ServerTenantLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = ServerTenantLabels.from_dict(_labels)

        phase = d.pop("phase", UNSET)

        resource_version = d.pop("resourceVersion", UNSET)

        def _parse_status(data: object) -> None | ServerTenantStatus | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                status_type_1 = ServerTenantStatus.from_dict(data)

                return status_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | ServerTenantStatus | Unset, data)

        status = _parse_status(d.pop("status", UNSET))

        server_tenant = cls(
            created_at=created_at,
            name=name,
            namespace=namespace,
            quotas=quotas,
            annotations=annotations,
            init_resources=init_resources,
            labels=labels,
            phase=phase,
            resource_version=resource_version,
            status=status,
        )

        server_tenant.additional_properties = d
        return server_tenant

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
