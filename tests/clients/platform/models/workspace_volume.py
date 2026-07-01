from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceVolume")


@_attrs_define
class WorkspaceVolume:
    """
    Example:
        {'mountPath': '/home/jovyan/work', 'name': 'notebook-data', 'size': '50Gi', 'storageClass': 'standard', 'used':
            '12Gi'}

    Attributes:
        mount_path (str): Path the volume is mounted at inside the container.
        name (str | Unset): Volume name; empty requests a new volume, set mounts an existing one.
        size (str | Unset): Requested capacity for a new volume (e.g. 50Gi).
        storage_class (str | Unset): StorageClass backing a new volume.
        used (str | Unset): Live consumed capacity of the volume (read-only).
    """

    mount_path: str
    name: str | Unset = UNSET
    size: str | Unset = UNSET
    storage_class: str | Unset = UNSET
    used: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        mount_path = self.mount_path

        name = self.name

        size = self.size

        storage_class = self.storage_class

        used = self.used

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "mountPath": mount_path,
            }
        )
        if name is not UNSET:
            field_dict["name"] = name
        if size is not UNSET:
            field_dict["size"] = size
        if storage_class is not UNSET:
            field_dict["storageClass"] = storage_class
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mount_path = d.pop("mountPath")

        name = d.pop("name", UNSET)

        size = d.pop("size", UNSET)

        storage_class = d.pop("storageClass", UNSET)

        used = d.pop("used", UNSET)

        workspace_volume = cls(
            mount_path=mount_path,
            name=name,
            size=size,
            storage_class=storage_class,
            used=used,
        )

        workspace_volume.additional_properties = d
        return workspace_volume

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
