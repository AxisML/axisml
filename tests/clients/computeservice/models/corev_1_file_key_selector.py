from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1FileKeySelector")


@_attrs_define
class Corev1FileKeySelector:
    """
    Attributes:
        key (str):
        path (str):
        volume_name (str):
        optional (bool | None | Unset):
    """

    key: str
    path: str
    volume_name: str
    optional: bool | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        key = self.key

        path = self.path

        volume_name = self.volume_name

        optional: bool | None | Unset
        if isinstance(self.optional, Unset):
            optional = UNSET
        else:
            optional = self.optional

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "key": key,
                "path": path,
                "volumeName": volume_name,
            }
        )
        if optional is not UNSET:
            field_dict["optional"] = optional

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        path = d.pop("path")

        volume_name = d.pop("volumeName")

        def _parse_optional(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        optional = _parse_optional(d.pop("optional", UNSET))

        corev_1_file_key_selector = cls(
            key=key,
            path=path,
            volume_name=volume_name,
            optional=optional,
        )

        corev_1_file_key_selector.additional_properties = d
        return corev_1_file_key_selector

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
