from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.data_volume import DataVolume


T = TypeVar("T", bound="DataVolumeList")


@_attrs_define
class DataVolumeList:
    """
    Example:
        {'count': 1, 'items': [{'accessModes': ['ReadWriteMany'], 'createdAt': '2026-06-20T08:00:00Z', 'description':
            'Shared raw datasets directory.', 'labels': {'team': 'vision'}, 'name': 'shared-datasets', 'size': '2Ti',
            'status': {'boundCapacity': '2Ti', 'mounts': [{'kind': 'Deployment', 'mountPath': '/data/shared', 'running':
            True, 'workload': 'ws-jupyter-3'}], 'phase': 'Bound'}, 'storageClass': 'nfs-rwx'}]}

    Attributes:
        count (int): Number of volumes in this page.
        items (list[DataVolume]): Data volumes in this page.
    """

    count: int
    items: list[DataVolume]
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
        from ..models.data_volume import DataVolume

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = DataVolume.from_dict(items_item_data)

            items.append(items_item)

        data_volume_list = cls(
            count=count,
            items=items,
        )

        data_volume_list.additional_properties = d
        return data_volume_list

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
