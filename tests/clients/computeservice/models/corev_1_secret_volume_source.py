from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_key_to_path import Corev1KeyToPath


T = TypeVar("T", bound="Corev1SecretVolumeSource")


@_attrs_define
class Corev1SecretVolumeSource:
    """
    Attributes:
        default_mode (int | None | Unset):
        items (list[Corev1KeyToPath] | Unset):
        optional (bool | None | Unset):
        secret_name (str | Unset):
    """

    default_mode: int | None | Unset = UNSET
    items: list[Corev1KeyToPath] | Unset = UNSET
    optional: bool | None | Unset = UNSET
    secret_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        default_mode: int | None | Unset
        if isinstance(self.default_mode, Unset):
            default_mode = UNSET
        else:
            default_mode = self.default_mode

        items: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.items, Unset):
            items = []
            for items_item_data in self.items:
                items_item = items_item_data.to_dict()
                items.append(items_item)

        optional: bool | None | Unset
        if isinstance(self.optional, Unset):
            optional = UNSET
        else:
            optional = self.optional

        secret_name = self.secret_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if default_mode is not UNSET:
            field_dict["defaultMode"] = default_mode
        if items is not UNSET:
            field_dict["items"] = items
        if optional is not UNSET:
            field_dict["optional"] = optional
        if secret_name is not UNSET:
            field_dict["secretName"] = secret_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_key_to_path import Corev1KeyToPath

        d = dict(src_dict)

        def _parse_default_mode(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        default_mode = _parse_default_mode(d.pop("defaultMode", UNSET))

        _items = d.pop("items", UNSET)
        items: list[Corev1KeyToPath] | Unset = UNSET
        if _items is not UNSET:
            items = []
            for items_item_data in _items:
                items_item = Corev1KeyToPath.from_dict(items_item_data)

                items.append(items_item)

        def _parse_optional(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        optional = _parse_optional(d.pop("optional", UNSET))

        secret_name = d.pop("secretName", UNSET)

        corev_1_secret_volume_source = cls(
            default_mode=default_mode,
            items=items,
            optional=optional,
            secret_name=secret_name,
        )

        corev_1_secret_volume_source.additional_properties = d
        return corev_1_secret_volume_source

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
