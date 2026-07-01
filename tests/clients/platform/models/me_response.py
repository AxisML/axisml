from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.user import User
    from ..models.user_tenant_role import UserTenantRole


T = TypeVar("T", bound="MeResponse")


@_attrs_define
class MeResponse:
    """
    Example:
        {'isSystemAdmin': False, 'permissions': ['job:read', 'job:write', 'run:read', 'run:trigger'], 'tenantRoles':
            [{'roleName': 'admin', 'tenantName': 'team-vision'}, {'roleName': 'member', 'tenantName': 'team-nlp'}], 'user':
            {'createdAt': '2026-06-20T08:00:00Z', 'disabled': False, 'displayName': 'Li Wei', 'email': 'li.wei@axisml.io',
            'id': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'mustChangePassword': False, 'updatedAt': '2026-06-28T09:30:00Z',
            'username': 'li.wei'}}

    Attributes:
        is_system_admin (bool): True if the caller holds the system administrator role.
        permissions (list[str]): Flattened permission strings resolved from the caller's roles.
        tenant_roles (list[UserTenantRole]): The caller's tenant/role bindings.
        user (User):  Example: {'createdAt': '2026-06-20T08:00:00Z', 'disabled': False, 'displayName': 'Li Wei',
            'email': 'li.wei@axisml.io', 'id': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'mustChangePassword': False,
            'updatedAt': '2026-06-28T09:30:00Z', 'username': 'li.wei'}.
    """

    is_system_admin: bool
    permissions: list[str]
    tenant_roles: list[UserTenantRole]
    user: User
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        is_system_admin = self.is_system_admin

        permissions = self.permissions

        tenant_roles = []
        for tenant_roles_item_data in self.tenant_roles:
            tenant_roles_item = tenant_roles_item_data.to_dict()
            tenant_roles.append(tenant_roles_item)

        user = self.user.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "isSystemAdmin": is_system_admin,
                "permissions": permissions,
                "tenantRoles": tenant_roles,
                "user": user,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.user import User
        from ..models.user_tenant_role import UserTenantRole

        d = dict(src_dict)
        is_system_admin = d.pop("isSystemAdmin")

        permissions = cast(list[str], d.pop("permissions"))

        tenant_roles = []
        _tenant_roles = d.pop("tenantRoles")
        for tenant_roles_item_data in _tenant_roles:
            tenant_roles_item = UserTenantRole.from_dict(tenant_roles_item_data)

            tenant_roles.append(tenant_roles_item)

        user = User.from_dict(d.pop("user"))

        me_response = cls(
            is_system_admin=is_system_admin,
            permissions=permissions,
            tenant_roles=tenant_roles,
            user=user,
        )

        me_response.additional_properties = d
        return me_response

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
