from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1AzureDiskVolumeSource")


@_attrs_define
class Corev1AzureDiskVolumeSource:
    """
    Attributes:
        disk_name (str):
        disk_uri (str):
        caching_mode (None | str | Unset):
        fs_type (None | str | Unset):
        kind (None | str | Unset):
        read_only (bool | None | Unset):
    """

    disk_name: str
    disk_uri: str
    caching_mode: None | str | Unset = UNSET
    fs_type: None | str | Unset = UNSET
    kind: None | str | Unset = UNSET
    read_only: bool | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        disk_name = self.disk_name

        disk_uri = self.disk_uri

        caching_mode: None | str | Unset
        if isinstance(self.caching_mode, Unset):
            caching_mode = UNSET
        else:
            caching_mode = self.caching_mode

        fs_type: None | str | Unset
        if isinstance(self.fs_type, Unset):
            fs_type = UNSET
        else:
            fs_type = self.fs_type

        kind: None | str | Unset
        if isinstance(self.kind, Unset):
            kind = UNSET
        else:
            kind = self.kind

        read_only: bool | None | Unset
        if isinstance(self.read_only, Unset):
            read_only = UNSET
        else:
            read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "diskName": disk_name,
                "diskURI": disk_uri,
            }
        )
        if caching_mode is not UNSET:
            field_dict["cachingMode"] = caching_mode
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if kind is not UNSET:
            field_dict["kind"] = kind
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disk_name = d.pop("diskName")

        disk_uri = d.pop("diskURI")

        def _parse_caching_mode(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        caching_mode = _parse_caching_mode(d.pop("cachingMode", UNSET))

        def _parse_fs_type(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        fs_type = _parse_fs_type(d.pop("fsType", UNSET))

        def _parse_kind(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        kind = _parse_kind(d.pop("kind", UNSET))

        def _parse_read_only(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        read_only = _parse_read_only(d.pop("readOnly", UNSET))

        corev_1_azure_disk_volume_source = cls(
            disk_name=disk_name,
            disk_uri=disk_uri,
            caching_mode=caching_mode,
            fs_type=fs_type,
            kind=kind,
            read_only=read_only,
        )

        corev_1_azure_disk_volume_source.additional_properties = d
        return corev_1_azure_disk_volume_source

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
