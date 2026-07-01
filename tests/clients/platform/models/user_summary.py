from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="UserSummary")


@_attrs_define
class UserSummary:
    """
    Example:
        {'displayName': 'Li Wei', 'email': 'li.wei@axisml.io', 'id': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'username':
            'li.wei'}

    Attributes:
        id (UUID): Stable user identifier.
        username (str): Unique account username.
        display_name (str | Unset): Human-readable display name.
        email (str | Unset): Account email address.
    """

    id: UUID
    username: str
    display_name: str | Unset = UNSET
    email: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = str(self.id)

        username = self.username

        display_name = self.display_name

        email = self.email

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "username": username,
            }
        )
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if email is not UNSET:
            field_dict["email"] = email

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = UUID(d.pop("id"))

        username = d.pop("username")

        display_name = d.pop("displayName", UNSET)

        email = d.pop("email", UNSET)

        user_summary = cls(
            id=id,
            username=username,
            display_name=display_name,
            email=email,
        )

        user_summary.additional_properties = d
        return user_summary

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
