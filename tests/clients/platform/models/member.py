from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.role_name import RoleName
from ..types import UNSET, Unset

T = TypeVar("T", bound="Member")


@_attrs_define
class Member:
    """
    Example:
        {'addedAt': '2026-06-20T08:00:00Z', 'displayName': 'Li Wei', 'email': 'li.wei@example.com', 'roleName': 'tenant-
            admin', 'userId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'username': 'li.wei'}

    Attributes:
        added_at (datetime.datetime): Time the member was added to the tenant.
        role_name (RoleName):
        user_id (UUID): User ID of the member.
        username (str): Username of the member.
        display_name (str | Unset): Human-readable name of the member.
        email (str | Unset): Email address of the member.
    """

    added_at: datetime.datetime
    role_name: RoleName
    user_id: UUID
    username: str
    display_name: str | Unset = UNSET
    email: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        added_at = self.added_at.isoformat()

        role_name = self.role_name.value

        user_id = str(self.user_id)

        username = self.username

        display_name = self.display_name

        email = self.email

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "addedAt": added_at,
                "roleName": role_name,
                "userId": user_id,
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
        added_at = datetime.datetime.fromisoformat(d.pop("addedAt"))

        role_name = RoleName(d.pop("roleName"))

        user_id = UUID(d.pop("userId"))

        username = d.pop("username")

        display_name = d.pop("displayName", UNSET)

        email = d.pop("email", UNSET)

        member = cls(
            added_at=added_at,
            role_name=role_name,
            user_id=user_id,
            username=username,
            display_name=display_name,
            email=email,
        )

        member.additional_properties = d
        return member

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
