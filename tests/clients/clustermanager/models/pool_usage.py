from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.server_resource_meter import ServerResourceMeter


T = TypeVar("T", bound="PoolUsage")


@_attrs_define
class PoolUsage:
    """
    Attributes:
        meters (list[ServerResourceMeter]): Per-resource used/total meters.
        pool (str): Resource pool name.
        tenant (str): Tenant identifier the usage is scoped to.
    """

    meters: list[ServerResourceMeter]
    pool: str
    tenant: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        meters = []
        for meters_item_data in self.meters:
            meters_item = meters_item_data.to_dict()
            meters.append(meters_item)

        pool = self.pool

        tenant = self.tenant

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "meters": meters,
                "pool": pool,
                "tenant": tenant,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_resource_meter import ServerResourceMeter

        d = dict(src_dict)
        meters = []
        _meters = d.pop("meters")
        for meters_item_data in _meters:
            meters_item = ServerResourceMeter.from_dict(meters_item_data)

            meters.append(meters_item)

        pool = d.pop("pool")

        tenant = d.pop("tenant")

        pool_usage = cls(
            meters=meters,
            pool=pool,
            tenant=tenant,
        )

        pool_usage.additional_properties = d
        return pool_usage

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
