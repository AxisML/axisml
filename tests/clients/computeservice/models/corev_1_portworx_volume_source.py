from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1PortworxVolumeSource")


@_attrs_define
class Corev1PortworxVolumeSource:
    """
    Attributes:
        volume_id (str):
        fs_type (str | Unset):
        read_only (bool | Unset):
    """

    volume_id: str
    fs_type: str | Unset = UNSET
    read_only: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        volume_id = self.volume_id

        fs_type = self.fs_type

        read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "volumeID": volume_id,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        volume_id = d.pop("volumeID")

        fs_type = d.pop("fsType", UNSET)

        read_only = d.pop("readOnly", UNSET)

        corev_1_portworx_volume_source = cls(
            volume_id=volume_id,
            fs_type=fs_type,
            read_only=read_only,
        )

        corev_1_portworx_volume_source.additional_properties = d
        return corev_1_portworx_volume_source

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
