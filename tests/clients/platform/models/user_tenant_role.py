from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.role_name import RoleName

T = TypeVar("T", bound="UserTenantRole")


@_attrs_define
class UserTenantRole:
    """
    Example:
        {'roleName': 'admin', 'tenantName': 'team-vision'}

    Attributes:
        role_name (RoleName):
        tenant_name (str): Tenant the role applies within.
    """

    role_name: RoleName
    tenant_name: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        role_name = self.role_name.value

        tenant_name = self.tenant_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "roleName": role_name,
                "tenantName": tenant_name,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        role_name = RoleName(d.pop("roleName"))

        tenant_name = d.pop("tenantName")

        user_tenant_role = cls(
            role_name=role_name,
            tenant_name=tenant_name,
        )

        user_tenant_role.additional_properties = d
        return user_tenant_role

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
