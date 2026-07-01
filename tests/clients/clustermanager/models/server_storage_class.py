from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ServerStorageClass")


@_attrs_define
class ServerStorageClass:
    """
    Attributes:
        allow_volume_expansion (bool): Whether volumes on this class can be expanded.
        default (bool): Whether this is the cluster default StorageClass.
        name (str): StorageClass name.
        provisioner (str | Unset): Provisioner backing the class.
    """

    allow_volume_expansion: bool
    default: bool
    name: str
    provisioner: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        allow_volume_expansion = self.allow_volume_expansion

        default = self.default

        name = self.name

        provisioner = self.provisioner

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "allowVolumeExpansion": allow_volume_expansion,
                "default": default,
                "name": name,
            }
        )
        if provisioner is not UNSET:
            field_dict["provisioner"] = provisioner

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        allow_volume_expansion = d.pop("allowVolumeExpansion")

        default = d.pop("default")

        name = d.pop("name")

        provisioner = d.pop("provisioner", UNSET)

        server_storage_class = cls(
            allow_volume_expansion=allow_volume_expansion,
            default=default,
            name=name,
            provisioner=provisioner,
        )

        server_storage_class.additional_properties = d
        return server_storage_class

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
