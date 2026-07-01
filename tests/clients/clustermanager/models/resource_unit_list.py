from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.server_resource_unit import ServerResourceUnit


T = TypeVar("T", bound="ResourceUnitList")


@_attrs_define
class ResourceUnitList:
    """
    Example:
        {'count': 1, 'items': [{'annotations': {'tenant.axisml.io/managed-by': 'platform'}, 'description': '2× A100 GPU
            compute unit.', 'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu':
            '2'}}]}

    Attributes:
        count (int): Number of units returned.
        items (list[ServerResourceUnit]): Resource units in the pool.
    """

    count: int
    items: list[ServerResourceUnit]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_resource_unit import ServerResourceUnit

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ServerResourceUnit.from_dict(items_item_data)

            items.append(items_item)

        resource_unit_list = cls(
            count=count,
            items=items,
        )

        resource_unit_list.additional_properties = d
        return resource_unit_list

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
