from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.user import User
    from ..models.user_tenant_role import UserTenantRole


T = TypeVar("T", bound="LoginResponse")


@_attrs_define
class LoginResponse:
    """
    Example:
        {'expiresAt': '2026-06-28T09:25:00Z', 'jwt': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJsaS53ZWkifQ.sig',
            'tenantRoles': [{'roleName': 'admin', 'tenantName': 'team-vision'}, {'roleName': 'member', 'tenantName': 'team-
            nlp'}], 'user': {'createdAt': '2026-06-20T08:00:00Z', 'disabled': False, 'displayName': 'Li Wei', 'email':
            'li.wei@axisml.io', 'id': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'mustChangePassword': False, 'updatedAt':
            '2026-06-28T09:30:00Z', 'username': 'li.wei'}}

    Attributes:
        jwt (str): Signed JWT bearer token for subsequent requests.
        tenant_roles (list[UserTenantRole]): Tenant/role bindings granted to the user.
        user (User):  Example: {'createdAt': '2026-06-20T08:00:00Z', 'disabled': False, 'displayName': 'Li Wei',
            'email': 'li.wei@axisml.io', 'id': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'mustChangePassword': False,
            'updatedAt': '2026-06-28T09:30:00Z', 'username': 'li.wei'}.
        expires_at (datetime.datetime | Unset): Time the token expires.
    """

    jwt: str
    tenant_roles: list[UserTenantRole]
    user: User
    expires_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        jwt = self.jwt

        tenant_roles = []
        for tenant_roles_item_data in self.tenant_roles:
            tenant_roles_item = tenant_roles_item_data.to_dict()
            tenant_roles.append(tenant_roles_item)

        user = self.user.to_dict()

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "jwt": jwt,
                "tenantRoles": tenant_roles,
                "user": user,
            }
        )
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.user import User
        from ..models.user_tenant_role import UserTenantRole

        d = dict(src_dict)
        jwt = d.pop("jwt")

        tenant_roles = []
        _tenant_roles = d.pop("tenantRoles")
        for tenant_roles_item_data in _tenant_roles:
            tenant_roles_item = UserTenantRole.from_dict(tenant_roles_item_data)

            tenant_roles.append(tenant_roles_item)

        user = User.from_dict(d.pop("user"))

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at, Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)

        login_response = cls(
            jwt=jwt,
            tenant_roles=tenant_roles,
            user=user,
            expires_at=expires_at,
        )

        login_response.additional_properties = d
        return login_response

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
