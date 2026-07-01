from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.quota_unit import QuotaUnit


T = TypeVar("T", bound="QuotaPatchRequest")


@_attrs_define
class QuotaPatchRequest:
    """
    Example:
        {'units': [{'quantity': 6, 'unitName': 'a100-2x'}]}

    Attributes:
        units (list[QuotaUnit]): Replacement per-unit allocations for the pool.
    """

    units: list[QuotaUnit]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        units = []
        for units_item_data in self.units:
            units_item = units_item_data.to_dict()
            units.append(units_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "units": units,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.quota_unit import QuotaUnit

        d = dict(src_dict)
        units = []
        _units = d.pop("units")
        for units_item_data in _units:
            units_item = QuotaUnit.from_dict(units_item_data)

            units.append(units_item)

        quota_patch_request = cls(
            units=units,
        )

        quota_patch_request.additional_properties = d
        return quota_patch_request

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
