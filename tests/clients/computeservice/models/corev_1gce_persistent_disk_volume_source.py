from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1GCEPersistentDiskVolumeSource")


@_attrs_define
class Corev1GCEPersistentDiskVolumeSource:
    """
    Attributes:
        pd_name (str):
        fs_type (str | Unset):
        partition (int | Unset):
        read_only (bool | Unset):
    """

    pd_name: str
    fs_type: str | Unset = UNSET
    partition: int | Unset = UNSET
    read_only: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pd_name = self.pd_name

        fs_type = self.fs_type

        partition = self.partition

        read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pdName": pd_name,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if partition is not UNSET:
            field_dict["partition"] = partition
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pd_name = d.pop("pdName")

        fs_type = d.pop("fsType", UNSET)

        partition = d.pop("partition", UNSET)

        read_only = d.pop("readOnly", UNSET)

        corev_1gce_persistent_disk_volume_source = cls(
            pd_name=pd_name,
            fs_type=fs_type,
            partition=partition,
            read_only=read_only,
        )

        corev_1gce_persistent_disk_volume_source.additional_properties = d
        return corev_1gce_persistent_disk_volume_source

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
