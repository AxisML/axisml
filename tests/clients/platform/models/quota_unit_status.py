from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="QuotaUnitStatus")


@_attrs_define
class QuotaUnitStatus:
    """
    Example:
        {'quantity': 4, 'unitName': 'a100-2x', 'used': 3}

    Attributes:
        quantity (int): Number of units allocated to the tenant.
        unit_name (str): Resource unit (shape) name.
        used (int | Unset): Number of units currently in use.
    """

    quantity: int
    unit_name: str
    used: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        quantity = self.quantity

        unit_name = self.unit_name

        used = self.used

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "quantity": quantity,
                "unitName": unit_name,
            }
        )
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        quantity = d.pop("quantity")

        unit_name = d.pop("unitName")

        used = d.pop("used", UNSET)

        quota_unit_status = cls(
            quantity=quantity,
            unit_name=unit_name,
            used=used,
        )

        quota_unit_status.additional_properties = d
        return quota_unit_status

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
