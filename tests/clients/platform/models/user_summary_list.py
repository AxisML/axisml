from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.user_summary import UserSummary


T = TypeVar("T", bound="UserSummaryList")


@_attrs_define
class UserSummaryList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'displayName': 'Li Wei', 'email': 'li.wei@axisml.io', 'id':
            '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'username': 'li.wei'}]}

    Attributes:
        count (int): Number of users in this page.
        items (list[UserSummary]): Users in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
    """

    count: int
    items: list[UserSummary]
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
        from ..models.user_summary import UserSummary

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = UserSummary.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        user_summary_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
        )

        user_summary_list.additional_properties = d
        return user_summary_list

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
