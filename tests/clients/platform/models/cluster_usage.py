from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.cluster_pool_usage import ClusterPoolUsage


T = TypeVar("T", bound="ClusterUsage")


@_attrs_define
class ClusterUsage:
    """
    Example:
        {'pools': [{'meters': [{'resource': 'gpu', 'total': 32, 'unit': 'cards', 'used': 22}, {'resource': 'cpu',
            'total': 384, 'unit': 'cores', 'used': 240}, {'resource': 'memory', 'total': 2048, 'unit': 'GiB', 'used':
            1228.8}], 'pool': 'gpu-a100'}, {'meters': [{'resource': 'gpu', 'total': 16, 'unit': 'cards', 'used': 14},
            {'resource': 'cpu', 'total': 256, 'unit': 'cores', 'used': 180}, {'resource': 'memory', 'total': 2048, 'unit':
            'GiB', 'used': 1433.6}], 'pool': 'h100-pool'}, {'meters': [{'resource': 'gpu', 'total': 0, 'unit': 'cards',
            'used': 0}, {'resource': 'cpu', 'total': 512, 'unit': 'cores', 'used': 320}, {'resource': 'memory', 'total':
            1536, 'unit': 'GiB', 'used': 819.2}], 'pool': 'cpu-pool'}], 'updatedAt': '2026-06-28T09:30:00Z'}

    Attributes:
        pools (list[ClusterPoolUsage]): Per-pool utilisation, one entry per pool the tenant has quota in.
        updated_at (datetime.datetime): Time the snapshot was sampled.
        partial (bool | Unset): True when one or more pools could not be sampled and were omitted from the snapshot.
    """

    pools: list[ClusterPoolUsage]
    updated_at: datetime.datetime
    partial: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pools = []
        for pools_item_data in self.pools:
            pools_item = pools_item_data.to_dict()
            pools.append(pools_item)

        updated_at = self.updated_at.isoformat()

        partial = self.partial

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pools": pools,
                "updatedAt": updated_at,
            }
        )
        if partial is not UNSET:
            field_dict["partial"] = partial

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.cluster_pool_usage import ClusterPoolUsage

        d = dict(src_dict)
        pools = []
        _pools = d.pop("pools")
        for pools_item_data in _pools:
            pools_item = ClusterPoolUsage.from_dict(pools_item_data)

            pools.append(pools_item)

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        partial = d.pop("partial", UNSET)

        cluster_usage = cls(
            pools=pools,
            updated_at=updated_at,
            partial=partial,
        )

        cluster_usage.additional_properties = d
        return cluster_usage

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
