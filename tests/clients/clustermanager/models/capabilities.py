from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="Capabilities")


@_attrs_define
class Capabilities:
    """
    Example:
        {'multiTenant': True, 'resourcePoolsWritable': True}

    Attributes:
        multi_tenant (bool): Whether Tenant CRUD is available (false = single static default tenant).
        resource_pools_writable (bool): Whether ResourcePool CRUD is available (false = single read-only default pool).
    """

    multi_tenant: bool
    resource_pools_writable: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        multi_tenant = self.multi_tenant

        resource_pools_writable = self.resource_pools_writable

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "multiTenant": multi_tenant,
                "resourcePoolsWritable": resource_pools_writable,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        multi_tenant = d.pop("multiTenant")

        resource_pools_writable = d.pop("resourcePoolsWritable")

        capabilities = cls(
            multi_tenant=multi_tenant,
            resource_pools_writable=resource_pools_writable,
        )

        capabilities.additional_properties = d
        return capabilities

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
