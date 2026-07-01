from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.member_create_request_role_name import MemberCreateRequestRoleName

T = TypeVar("T", bound="MemberCreateRequest")


@_attrs_define
class MemberCreateRequest:
    """
    Example:
        {'account': 'zhang.san@example.com', 'roleName': 'user'}

    Attributes:
        account (str): Email or username of the existing platform user to add.
        role_name (MemberCreateRequestRoleName): Role to grant the member (tenant-admin or user).
    """

    account: str
    role_name: MemberCreateRequestRoleName
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        account = self.account

        role_name = self.role_name.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "account": account,
                "roleName": role_name,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        account = d.pop("account")

        role_name = MemberCreateRequestRoleName(d.pop("roleName"))

        member_create_request = cls(
            account=account,
            role_name=role_name,
        )

        member_create_request.additional_properties = d
        return member_create_request

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
