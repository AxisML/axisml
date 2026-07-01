from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.config_map_source_ref import ConfigMapSourceRef


T = TypeVar("T", bound="ConfigMapInit")


@_attrs_define
class ConfigMapInit:
    """
    Example:
        {'name': 'shared-config', 'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-system'}}

    Attributes:
        name (str): Name of the ConfigMap to create in the tenant namespace.
        source_config_map_ref (ConfigMapSourceRef):  Example: {'name': 'shared-config', 'namespace': 'axisml-system'}.
    """

    name: str
    source_config_map_ref: ConfigMapSourceRef
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
        from ..models.config_map_source_ref import ConfigMapSourceRef

        d = dict(src_dict)
        name = d.pop("name")

        source_config_map_ref = ConfigMapSourceRef.from_dict(
            d.pop("sourceConfigMapRef")
        )

        config_map_init = cls(
            name=name,
            source_config_map_ref=source_config_map_ref,
        )

        config_map_init.additional_properties = d
        return config_map_init

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
