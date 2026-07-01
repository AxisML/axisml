from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1GlusterfsVolumeSource")


@_attrs_define
class Corev1GlusterfsVolumeSource:
    """
    Attributes:
        endpoints (str):
        path (str):
        read_only (bool | Unset):
    """

    endpoints: str
    path: str
    read_only: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        endpoints = self.endpoints

        path = self.path

        read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "endpoints": endpoints,
                "path": path,
            }
        )
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        endpoints = d.pop("endpoints")

        path = d.pop("path")

        read_only = d.pop("readOnly", UNSET)

        corev_1_glusterfs_volume_source = cls(
            endpoints=endpoints,
            path=path,
            read_only=read_only,
        )

        corev_1_glusterfs_volume_source.additional_properties = d
        return corev_1_glusterfs_volume_source

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
