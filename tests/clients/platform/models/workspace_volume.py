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
        {'mountPath': '/home/jovyan/work', 'name': 'notebook-data', 'used': '12Gi'}

    Attributes:
        mount_path (str): Path the volume is mounted at inside the container.
        name (str): Existing data volume (claim) name to mount.
        used (str | Unset): Live consumed capacity of the volume (read-only).
    """

    mount_path: str
    name: str
    used: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        mount_path = self.mount_path

        name = self.name

        used = self.used

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "mountPath": mount_path,
                "name": name,
            }
        )
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mount_path = d.pop("mountPath")

        name = d.pop("name")

        used = d.pop("used", UNSET)

        workspace_volume = cls(
            mount_path=mount_path,
            name=name,
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
