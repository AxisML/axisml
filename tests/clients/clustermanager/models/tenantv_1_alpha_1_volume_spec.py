from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Tenantv1Alpha1VolumeSpec")


@_attrs_define
class Tenantv1Alpha1VolumeSpec:
    """
    Attributes:
        name (str):
        access_modes (list[str] | Unset):
        description (str | Unset):
        host_path (str | Unset):
        size (str | Unset):
        storage_class (str | Unset):
    """

    name: str
    access_modes: list[str] | Unset = UNSET
    description: str | Unset = UNSET
    host_path: str | Unset = UNSET
    size: str | Unset = UNSET
    storage_class: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        access_modes: list[str] | Unset = UNSET
        if not isinstance(self.access_modes, Unset):
            access_modes = self.access_modes

        description = self.description

        host_path = self.host_path

        size = self.size

        storage_class = self.storage_class

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if access_modes is not UNSET:
            field_dict["accessModes"] = access_modes
        if description is not UNSET:
            field_dict["description"] = description
        if host_path is not UNSET:
            field_dict["hostPath"] = host_path
        if size is not UNSET:
            field_dict["size"] = size
        if storage_class is not UNSET:
            field_dict["storageClass"] = storage_class

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        access_modes = cast(list[str], d.pop("accessModes", UNSET))

        description = d.pop("description", UNSET)

        host_path = d.pop("hostPath", UNSET)

        size = d.pop("size", UNSET)

        storage_class = d.pop("storageClass", UNSET)

        tenantv_1_alpha_1_volume_spec = cls(
            name=name,
            access_modes=access_modes,
            description=description,
            host_path=host_path,
            size=size,
            storage_class=storage_class,
        )

        tenantv_1_alpha_1_volume_spec.additional_properties = d
        return tenantv_1_alpha_1_volume_spec

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
