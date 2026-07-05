from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.cluster_meter import ClusterMeter


T = TypeVar("T", bound="ClusterPoolUsage")


@_attrs_define
class ClusterPoolUsage:
    """
    Example:
        {'meters': [{'resource': 'gpu', 'total': 32, 'unit': 'cards', 'used': 22}, {'resource': 'cpu', 'total': 384,
            'unit': 'cores', 'used': 240}, {'resource': 'memory', 'total': 2048, 'unit': 'GiB', 'used': 1228.8}], 'pool':
            'gpu-a100'}

    Attributes:
        meters (list[ClusterMeter]): Per-resource utilisation meters for the pool.
        pool (str): Resource pool name.
    """

    meters: list[ClusterMeter]
    pool: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        meters = []
        for meters_item_data in self.meters:
            meters_item = meters_item_data.to_dict()
            meters.append(meters_item)

        pool = self.pool

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "meters": meters,
                "pool": pool,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.cluster_meter import ClusterMeter

        d = dict(src_dict)
        meters = []
        _meters = d.pop("meters")
        for meters_item_data in _meters:
            meters_item = ClusterMeter.from_dict(meters_item_data)

            meters.append(meters_item)

        pool = d.pop("pool")

        cluster_pool_usage = cls(
            meters=meters,
            pool=pool,
        )

        cluster_pool_usage.additional_properties = d
        return cluster_pool_usage

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
