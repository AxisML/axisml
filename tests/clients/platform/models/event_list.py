from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.event import Event


T = TypeVar("T", bound="EventList")


@_attrs_define
class EventList:
    """
    Example:
        {'count': 1, 'items': [{'count': 1, 'firstTimestamp': '2026-06-28T09:00:00Z', 'involvedObject': {'kind': 'Pod',
            'name': 'resnet-train-7-worker-0', 'namespace': 'axisml-team-vision'}, 'lastTimestamp': '2026-06-28T09:00:00Z',
            'message': 'Successfully assigned resnet-train-7-worker-0 to gpu-node-03.', 'reason': 'Scheduled', 'source':
            'default-scheduler', 'type': 'Normal'}]}

    Attributes:
        count (int): Number of events in the list.
        items (list[Event]): Events in the list.
    """

    count: int
    items: list[Event]
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
        from ..models.event import Event

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Event.from_dict(items_item_data)

            items.append(items_item)

        event_list = cls(
            count=count,
            items=items,
        )

        event_list.additional_properties = d
        return event_list

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
