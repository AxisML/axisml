from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1FCVolumeSource")


@_attrs_define
class Corev1FCVolumeSource:
    """
    Attributes:
        fs_type (str | Unset):
        lun (int | None | Unset):
        read_only (bool | Unset):
        target_ww_ns (list[str] | Unset):
        wwids (list[str] | Unset):
    """

    fs_type: str | Unset = UNSET
    lun: int | None | Unset = UNSET
    read_only: bool | Unset = UNSET
    target_ww_ns: list[str] | Unset = UNSET
    wwids: list[str] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        fs_type = self.fs_type

        lun: int | None | Unset
        if isinstance(self.lun, Unset):
            lun = UNSET
        else:
            lun = self.lun

        read_only = self.read_only

        target_ww_ns: list[str] | Unset = UNSET
        if not isinstance(self.target_ww_ns, Unset):
            target_ww_ns = self.target_ww_ns

        wwids: list[str] | Unset = UNSET
        if not isinstance(self.wwids, Unset):
            wwids = self.wwids

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if lun is not UNSET:
            field_dict["lun"] = lun
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if target_ww_ns is not UNSET:
            field_dict["targetWWNs"] = target_ww_ns
        if wwids is not UNSET:
            field_dict["wwids"] = wwids

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fs_type = d.pop("fsType", UNSET)

        def _parse_lun(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        lun = _parse_lun(d.pop("lun", UNSET))

        read_only = d.pop("readOnly", UNSET)

        target_ww_ns = cast(list[str], d.pop("targetWWNs", UNSET))

        wwids = cast(list[str], d.pop("wwids", UNSET))

        corev_1fc_volume_source = cls(
            fs_type=fs_type,
            lun=lun,
            read_only=read_only,
            target_ww_ns=target_ww_ns,
            wwids=wwids,
        )

        corev_1fc_volume_source.additional_properties = d
        return corev_1fc_volume_source

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
