from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1PhotonPersistentDiskVolumeSource")


@_attrs_define
class Corev1PhotonPersistentDiskVolumeSource:
    """
    Attributes:
        pd_id (str):
        fs_type (str | Unset):
    """

    pd_id: str
    fs_type: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pd_id = self.pd_id

        fs_type = self.fs_type

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pdID": pd_id,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pd_id = d.pop("pdID")

        fs_type = d.pop("fsType", UNSET)

        corev_1_photon_persistent_disk_volume_source = cls(
            pd_id=pd_id,
            fs_type=fs_type,
        )

        corev_1_photon_persistent_disk_volume_source.additional_properties = d
        return corev_1_photon_persistent_disk_volume_source

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
