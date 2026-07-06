from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ResourceMeter")


@_attrs_define
class ResourceMeter:
    """
    Attributes:
        resource (str): Resource dimension (cpu, memory, nvidia.com/gpu).
        total (float): The tenant's quota ceiling in this pool.
        used (float): Amount currently in use (from the ElasticQuota status).
        unit (str | Unset): Value unit (cores, GiB, cards).
    """

    resource: str
    total: float
    used: float
    unit: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        resource = self.resource

        total = self.total

        used = self.used

        unit = self.unit

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "resource": resource,
                "total": total,
                "used": used,
            }
        )
        if unit is not UNSET:
            field_dict["unit"] = unit

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        resource = d.pop("resource")

        total = d.pop("total")

        used = d.pop("used")

        unit = d.pop("unit", UNSET)

        resource_meter = cls(
            resource=resource,
            total=total,
            used=used,
            unit=unit,
        )

        resource_meter.additional_properties = d
        return resource_meter

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
