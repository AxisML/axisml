from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1TypedLocalObjectReference")


@_attrs_define
class Corev1TypedLocalObjectReference:
    """
    Attributes:
        kind (str):
        name (str):
        api_group (None | str | Unset):
    """

    kind: str
    name: str
    api_group: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        name = self.name

        api_group: None | str | Unset
        if isinstance(self.api_group, Unset):
            api_group = UNSET
        else:
            api_group = self.api_group

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
                "name": name,
            }
        )
        if api_group is not UNSET:
            field_dict["apiGroup"] = api_group

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        name = d.pop("name")

        def _parse_api_group(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        api_group = _parse_api_group(d.pop("apiGroup", UNSET))

        corev_1_typed_local_object_reference = cls(
            kind=kind,
            name=name,
            api_group=api_group,
        )

        corev_1_typed_local_object_reference.additional_properties = d
        return corev_1_typed_local_object_reference

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
