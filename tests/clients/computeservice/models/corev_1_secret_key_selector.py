from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="Corev1SecretKeySelector")


@_attrs_define
class Corev1SecretKeySelector:
    """
    Attributes:
        local_object_reference (Corev1LocalObjectReference):
        key (str):
        optional (bool | None | Unset):
    """

    local_object_reference: Corev1LocalObjectReference
    key: str
    optional: bool | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        local_object_reference = self.local_object_reference.to_dict()

        key = self.key

        optional: bool | None | Unset
        if isinstance(self.optional, Unset):
            optional = UNSET
        else:
            optional = self.optional

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "LocalObjectReference": local_object_reference,
                "key": key,
            }
        )
        if optional is not UNSET:
            field_dict["optional"] = optional

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        local_object_reference = Corev1LocalObjectReference.from_dict(
            d.pop("LocalObjectReference")
        )

        key = d.pop("key")

        def _parse_optional(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        optional = _parse_optional(d.pop("optional", UNSET))

        corev_1_secret_key_selector = cls(
            local_object_reference=local_object_reference,
            key=key,
            optional=optional,
        )

        corev_1_secret_key_selector.additional_properties = d
        return corev_1_secret_key_selector

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
