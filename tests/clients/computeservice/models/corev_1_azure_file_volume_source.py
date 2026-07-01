from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1AzureFileVolumeSource")


@_attrs_define
class Corev1AzureFileVolumeSource:
    """
    Attributes:
        secret_name (str):
        share_name (str):
        read_only (bool | Unset):
    """

    secret_name: str
    share_name: str
    read_only: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        secret_name = self.secret_name

        share_name = self.share_name

        read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "secretName": secret_name,
                "shareName": share_name,
            }
        )
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        secret_name = d.pop("secretName")

        share_name = d.pop("shareName")

        read_only = d.pop("readOnly", UNSET)

        corev_1_azure_file_volume_source = cls(
            secret_name=secret_name,
            share_name=share_name,
            read_only=read_only,
        )

        corev_1_azure_file_volume_source.additional_properties = d
        return corev_1_azure_file_volume_source

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
