from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.tenantv_1_alpha_1_source_config_map_ref import (
        Tenantv1Alpha1SourceConfigMapRef,
    )


T = TypeVar("T", bound="Tenantv1Alpha1ConfigMapSpec")


@_attrs_define
class Tenantv1Alpha1ConfigMapSpec:
    """
    Attributes:
        name (str):
        source_config_map_ref (Tenantv1Alpha1SourceConfigMapRef):
    """

    name: str
    source_config_map_ref: Tenantv1Alpha1SourceConfigMapRef
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        source_config_map_ref = self.source_config_map_ref.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "sourceConfigMapRef": source_config_map_ref,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenantv_1_alpha_1_source_config_map_ref import (
            Tenantv1Alpha1SourceConfigMapRef,
        )

        d = dict(src_dict)
        name = d.pop("name")

        source_config_map_ref = Tenantv1Alpha1SourceConfigMapRef.from_dict(
            d.pop("sourceConfigMapRef")
        )

        tenantv_1_alpha_1_config_map_spec = cls(
            name=name,
            source_config_map_ref=source_config_map_ref,
        )

        tenantv_1_alpha_1_config_map_spec.additional_properties = d
        return tenantv_1_alpha_1_config_map_spec

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
