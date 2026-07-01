from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1QuobyteVolumeSource")


@_attrs_define
class Corev1QuobyteVolumeSource:
    """
    Attributes:
        registry (str):
        volume (str):
        group (str | Unset):
        read_only (bool | Unset):
        tenant (str | Unset):
        user (str | Unset):
    """

    registry: str
    volume: str
    group: str | Unset = UNSET
    read_only: bool | Unset = UNSET
    tenant: str | Unset = UNSET
    user: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        registry = self.registry

        volume = self.volume

        group = self.group

        read_only = self.read_only

        tenant = self.tenant

        user = self.user

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "registry": registry,
                "volume": volume,
            }
        )
        if group is not UNSET:
            field_dict["group"] = group
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if tenant is not UNSET:
            field_dict["tenant"] = tenant
        if user is not UNSET:
            field_dict["user"] = user

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        registry = d.pop("registry")

        volume = d.pop("volume")

        group = d.pop("group", UNSET)

        read_only = d.pop("readOnly", UNSET)

        tenant = d.pop("tenant", UNSET)

        user = d.pop("user", UNSET)

        corev_1_quobyte_volume_source = cls(
            registry=registry,
            volume=volume,
            group=group,
            read_only=read_only,
            tenant=tenant,
            user=user,
        )

        corev_1_quobyte_volume_source.additional_properties = d
        return corev_1_quobyte_volume_source

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
