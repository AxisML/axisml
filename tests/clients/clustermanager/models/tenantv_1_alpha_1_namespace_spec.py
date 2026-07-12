from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenantv_1_alpha_1_namespace_spec_annotations import (
        Tenantv1Alpha1NamespaceSpecAnnotations,
    )
    from ..models.tenantv_1_alpha_1_namespace_spec_labels import (
        Tenantv1Alpha1NamespaceSpecLabels,
    )


T = TypeVar("T", bound="Tenantv1Alpha1NamespaceSpec")


@_attrs_define
class Tenantv1Alpha1NamespaceSpec:
    """
    Attributes:
        name (str):
        annotations (Tenantv1Alpha1NamespaceSpecAnnotations | Unset):
        labels (Tenantv1Alpha1NamespaceSpecLabels | Unset):
    """

    name: str
    annotations: Tenantv1Alpha1NamespaceSpecAnnotations | Unset = UNSET
    labels: Tenantv1Alpha1NamespaceSpecLabels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if labels is not UNSET:
            field_dict["labels"] = labels

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenantv_1_alpha_1_namespace_spec_annotations import (
            Tenantv1Alpha1NamespaceSpecAnnotations,
        )
        from ..models.tenantv_1_alpha_1_namespace_spec_labels import (
            Tenantv1Alpha1NamespaceSpecLabels,
        )

        d = dict(src_dict)
        name = d.pop("name")

        _annotations = d.pop("annotations", UNSET)
        annotations: Tenantv1Alpha1NamespaceSpecAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = Tenantv1Alpha1NamespaceSpecAnnotations.from_dict(_annotations)

        _labels = d.pop("labels", UNSET)
        labels: Tenantv1Alpha1NamespaceSpecLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = Tenantv1Alpha1NamespaceSpecLabels.from_dict(_labels)

        tenantv_1_alpha_1_namespace_spec = cls(
            name=name,
            annotations=annotations,
            labels=labels,
        )

        tenantv_1_alpha_1_namespace_spec.additional_properties = d
        return tenantv_1_alpha_1_namespace_spec

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
