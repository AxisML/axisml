from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.server_quota_unit import ServerQuotaUnit


T = TypeVar("T", bound="ServerQuota")


@_attrs_define
class ServerQuota:
    """
    Attributes:
        pool (str): ResourcePool this quota applies to.
        units (list[ServerQuotaUnit]): Unit × quantity selections granted to the tenant under this pool.
    """

    pool: str
    units: list[ServerQuotaUnit]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pool = self.pool

        units = []
        for units_item_data in self.units:
            units_item = units_item_data.to_dict()
            units.append(units_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pool": pool,
                "units": units,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota_unit import ServerQuotaUnit

        d = dict(src_dict)
        pool = d.pop("pool")

        units = []
        _units = d.pop("units")
        for units_item_data in _units:
            units_item = ServerQuotaUnit.from_dict(units_item_data)

            units.append(units_item)

        server_quota = cls(
            pool=pool,
            units=units,
        )

        server_quota.additional_properties = d
        return server_quota

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
