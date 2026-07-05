from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.activity_item import ActivityItem


T = TypeVar("T", bound="ActivityList")


@_attrs_define
class ActivityList:
    """
    Example:
        {'count': 3, 'items': [{'action': 'succeeded', 'actor': 'li.wei', 'id': 'act-9f3a2e5c', 'kind': 'run', 'name':
            'resnet-train-7', 'phase': 'Succeeded', 'timestamp': '2026-06-28T09:25:00Z'}, {'action': 'started', 'actor':
            'zhang.san', 'id': 'act-7b4f2a1c', 'kind': 'mlservice', 'name': 'llama3-chat', 'phase': 'Ready', 'timestamp':
            '2026-06-28T09:30:00Z'}, {'action': 'created', 'actor': 'li.wei', 'id': 'act-3e8f4a1c', 'kind': 'workspace',
            'name': 'notebook-dev', 'phase': 'Running', 'timestamp': '2026-06-20T08:00:00Z'}]}

    Attributes:
        count (int): Number of entries in this page.
        items (list[ActivityItem]): Activity entries, newest first.
    """

    count: int
    items: list[ActivityItem]
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
        from ..models.activity_item import ActivityItem

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ActivityItem.from_dict(items_item_data)

            items.append(items_item)

        activity_list = cls(
            count=count,
            items=items,
        )

        activity_list.additional_properties = d
        return activity_list

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
