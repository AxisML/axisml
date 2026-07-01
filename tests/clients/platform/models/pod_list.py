from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.pod import Pod


T = TypeVar("T", bound="PodList")


@_attrs_define
class PodList:
    """
    Example:
        {'count': 1, 'items': [{'name': 'resnet-train-7-worker-0', 'nodeName': 'gpu-node-03', 'phase': 'Running',
            'replicaIndex': 0, 'restartCount': 0, 'role': 'worker', 'startedAt': '2026-06-28T09:00:00Z'}]}

    Attributes:
        count (int): Number of pods in the list.
        items (list[Pod]): Pods in the list.
    """

    count: int
    items: list[Pod]
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
        from ..models.pod import Pod

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Pod.from_dict(items_item_data)

            items.append(items_item)

        pod_list = cls(
            count=count,
            items=items,
        )

        pod_list.additional_properties = d
        return pod_list

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
