from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.quota_unit_status import QuotaUnitStatus


T = TypeVar("T", bound="QuotaStatus")


@_attrs_define
class QuotaStatus:
    """
    Example:
        {'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x', 'used': 3}]}

    Attributes:
        pool (str): Resource pool the status refers to.
        units (list[QuotaUnitStatus] | Unset): Live per-unit usage within the pool.
    """

    pool: str
    units: list[QuotaUnitStatus] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pool = self.pool

        units: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.units, Unset):
            units = []
            for units_item_data in self.units:
                units_item = units_item_data.to_dict()
                units.append(units_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pool": pool,
            }
        )
        if units is not UNSET:
            field_dict["units"] = units

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.quota_unit_status import QuotaUnitStatus

        d = dict(src_dict)
        pool = d.pop("pool")

        _units = d.pop("units", UNSET)
        units: list[QuotaUnitStatus] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = QuotaUnitStatus.from_dict(units_item_data)

                units.append(units_item)

        quota_status = cls(
            pool=pool,
            units=units,
        )

        quota_status.additional_properties = d
        return quota_status

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
