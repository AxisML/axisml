from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resource_unit import ResourceUnit


T = TypeVar("T", bound="ResourceUnitList")


@_attrs_define
class ResourceUnitList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'annotations': {'axisml.io/cost-center': 'ml-platform',
            'axisml.io/created-by': 'admin'}, 'description': '2x A100 GPU compute unit.', 'limits': {'cpu': '16', 'memory':
            '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x', 'nodeSelector': {'arch': 'amd64', 'gpu.product': 'A100'},
            'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'tolerations': [{'effect': 'NoSchedule',
            'key': 'nvidia.com/gpu', 'operator': 'Exists'}]}]}

    Attributes:
        count (int): Number of units in this page.
        items (list[ResourceUnit]): Resource units in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
    """

    count: int
    items: list[ResourceUnit]
    continue_token: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        continue_token = self.continue_token

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
            }
        )
        if continue_token is not UNSET:
            field_dict["continueToken"] = continue_token

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resource_unit import ResourceUnit

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ResourceUnit.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        resource_unit_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
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
