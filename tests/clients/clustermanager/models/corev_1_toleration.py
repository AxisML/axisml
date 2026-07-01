from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1Toleration")


@_attrs_define
class Corev1Toleration:
    """
    Attributes:
        effect (str | Unset):
        key (str | Unset):
        operator (str | Unset):
        toleration_seconds (int | None | Unset):
        value (str | Unset):
    """

    effect: str | Unset = UNSET
    key: str | Unset = UNSET
    operator: str | Unset = UNSET
    toleration_seconds: int | None | Unset = UNSET
    value: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        effect = self.effect

        key = self.key

        operator = self.operator

        toleration_seconds: int | None | Unset
        if isinstance(self.toleration_seconds, Unset):
            toleration_seconds = UNSET
        else:
            toleration_seconds = self.toleration_seconds

        value = self.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if effect is not UNSET:
            field_dict["effect"] = effect
        if key is not UNSET:
            field_dict["key"] = key
        if operator is not UNSET:
            field_dict["operator"] = operator
        if toleration_seconds is not UNSET:
            field_dict["tolerationSeconds"] = toleration_seconds
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        effect = d.pop("effect", UNSET)

        key = d.pop("key", UNSET)

        operator = d.pop("operator", UNSET)

        def _parse_toleration_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        toleration_seconds = _parse_toleration_seconds(
            d.pop("tolerationSeconds", UNSET)
        )

        value = d.pop("value", UNSET)

        corev_1_toleration = cls(
            effect=effect,
            key=key,
            operator=operator,
            toleration_seconds=toleration_seconds,
            value=value,
        )

        corev_1_toleration.additional_properties = d
        return corev_1_toleration

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
