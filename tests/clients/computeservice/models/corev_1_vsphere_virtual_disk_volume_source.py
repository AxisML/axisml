from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1VsphereVirtualDiskVolumeSource")


@_attrs_define
class Corev1VsphereVirtualDiskVolumeSource:
    """
    Attributes:
        volume_path (str):
        fs_type (str | Unset):
        storage_policy_id (str | Unset):
        storage_policy_name (str | Unset):
    """

    volume_path: str
    fs_type: str | Unset = UNSET
    storage_policy_id: str | Unset = UNSET
    storage_policy_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        volume_path = self.volume_path

        fs_type = self.fs_type

        storage_policy_id = self.storage_policy_id

        storage_policy_name = self.storage_policy_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "volumePath": volume_path,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if storage_policy_id is not UNSET:
            field_dict["storagePolicyID"] = storage_policy_id
        if storage_policy_name is not UNSET:
            field_dict["storagePolicyName"] = storage_policy_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        volume_path = d.pop("volumePath")

        fs_type = d.pop("fsType", UNSET)

        storage_policy_id = d.pop("storagePolicyID", UNSET)

        storage_policy_name = d.pop("storagePolicyName", UNSET)

        corev_1_vsphere_virtual_disk_volume_source = cls(
            volume_path=volume_path,
            fs_type=fs_type,
            storage_policy_id=storage_policy_id,
            storage_policy_name=storage_policy_name,
        )

        corev_1_vsphere_virtual_disk_volume_source.additional_properties = d
        return corev_1_vsphere_virtual_disk_volume_source

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
