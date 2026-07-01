from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.quota import Quota
    from ..models.quota_status import QuotaStatus


T = TypeVar("T", bound="QuotaList")


@_attrs_define
class QuotaList:
    """
    Example:
        {'count': 1, 'items': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}], 'statuses':
            [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x', 'used': 3}]}]}

    Attributes:
        count (int): Number of pool quotas in this list.
        items (list[Quota]): Per-pool quota allocations for the tenant.
        statuses (list[QuotaStatus] | Unset): Live per-pool usage matching the items.
    """

    count: int
    items: list[Quota]
    statuses: list[QuotaStatus] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        statuses: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.statuses, Unset):
            statuses = []
            for statuses_item_data in self.statuses:
                statuses_item = statuses_item_data.to_dict()
                statuses.append(statuses_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
            }
        )
        if statuses is not UNSET:
            field_dict["statuses"] = statuses

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.quota import Quota
        from ..models.quota_status import QuotaStatus

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Quota.from_dict(items_item_data)

            items.append(items_item)

        _statuses = d.pop("statuses", UNSET)
        statuses: list[QuotaStatus] | Unset = UNSET
        if _statuses is not UNSET:
            statuses = []
            for statuses_item_data in _statuses:
                statuses_item = QuotaStatus.from_dict(statuses_item_data)

                statuses.append(statuses_item)

        quota_list = cls(
            count=count,
            items=items,
            statuses=statuses,
        )

        quota_list.additional_properties = d
        return quota_list

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
