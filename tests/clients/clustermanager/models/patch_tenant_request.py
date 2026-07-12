from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.patch_tenant_request_annotations import PatchTenantRequestAnnotations
    from ..models.patch_tenant_request_labels import PatchTenantRequestLabels
    from ..models.patch_tenant_request_namespace_annotations import (
        PatchTenantRequestNamespaceAnnotations,
    )
    from ..models.patch_tenant_request_namespace_labels import (
        PatchTenantRequestNamespaceLabels,
    )
    from ..models.tenantv_1_alpha_1_init_resources import Tenantv1Alpha1InitResources


T = TypeVar("T", bound="PatchTenantRequest")


@_attrs_define
class PatchTenantRequest:
    """
    Example:
        {'labels': {'displayName': 'Vision Team', 'region': 'cn-east'}, 'namespaceLabels': {'team': 'vision'}}

    Attributes:
        annotations (PatchTenantRequestAnnotations | Unset): Replacement annotations for the tenant.
        init_resources (None | Tenantv1Alpha1InitResources | Unset): Replacement per-tenant init resources.
        labels (PatchTenantRequestLabels | Unset): Replacement labels for the tenant.
        namespace_annotations (PatchTenantRequestNamespaceAnnotations | Unset): Replacement annotations applied to the
            tenant's namespace.
        namespace_labels (PatchTenantRequestNamespaceLabels | Unset): Replacement labels applied to the tenant's
            namespace.
    """

    annotations: PatchTenantRequestAnnotations | Unset = UNSET
    init_resources: None | Tenantv1Alpha1InitResources | Unset = UNSET
    labels: PatchTenantRequestLabels | Unset = UNSET
    namespace_annotations: PatchTenantRequestNamespaceAnnotations | Unset = UNSET
    namespace_labels: PatchTenantRequestNamespaceLabels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
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

        namespace_annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.namespace_annotations, Unset):
            namespace_annotations = self.namespace_annotations.to_dict()

        namespace_labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.namespace_labels, Unset):
            namespace_labels = self.namespace_labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if init_resources is not UNSET:
            field_dict["initResources"] = init_resources
        if labels is not UNSET:
            field_dict["labels"] = labels
        if namespace_annotations is not UNSET:
            field_dict["namespaceAnnotations"] = namespace_annotations
        if namespace_labels is not UNSET:
            field_dict["namespaceLabels"] = namespace_labels

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.patch_tenant_request_annotations import (
            PatchTenantRequestAnnotations,
        )
        from ..models.patch_tenant_request_labels import PatchTenantRequestLabels
        from ..models.patch_tenant_request_namespace_annotations import (
            PatchTenantRequestNamespaceAnnotations,
        )
        from ..models.patch_tenant_request_namespace_labels import (
            PatchTenantRequestNamespaceLabels,
        )
        from ..models.tenantv_1_alpha_1_init_resources import (
            Tenantv1Alpha1InitResources,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: PatchTenantRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = PatchTenantRequestAnnotations.from_dict(_annotations)

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
        labels: PatchTenantRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = PatchTenantRequestLabels.from_dict(_labels)

        _namespace_annotations = d.pop("namespaceAnnotations", UNSET)
        namespace_annotations: PatchTenantRequestNamespaceAnnotations | Unset
        if isinstance(_namespace_annotations, Unset):
            namespace_annotations = UNSET
        else:
            namespace_annotations = PatchTenantRequestNamespaceAnnotations.from_dict(
                _namespace_annotations
            )

        _namespace_labels = d.pop("namespaceLabels", UNSET)
        namespace_labels: PatchTenantRequestNamespaceLabels | Unset
        if isinstance(_namespace_labels, Unset):
            namespace_labels = UNSET
        else:
            namespace_labels = PatchTenantRequestNamespaceLabels.from_dict(
                _namespace_labels
            )

        patch_tenant_request = cls(
            annotations=annotations,
            init_resources=init_resources,
            labels=labels,
            namespace_annotations=namespace_annotations,
            namespace_labels=namespace_labels,
        )

        patch_tenant_request.additional_properties = d
        return patch_tenant_request

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
