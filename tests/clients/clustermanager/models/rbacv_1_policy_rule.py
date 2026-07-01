from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Rbacv1PolicyRule")


@_attrs_define
class Rbacv1PolicyRule:
    """
    Attributes:
        verbs (list[str]):
        api_groups (list[str] | Unset):
        non_resource_ur_ls (list[str] | Unset):
        resource_names (list[str] | Unset):
        resources (list[str] | Unset):
    """

    verbs: list[str]
    api_groups: list[str] | Unset = UNSET
    non_resource_ur_ls: list[str] | Unset = UNSET
    resource_names: list[str] | Unset = UNSET
    resources: list[str] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        verbs = self.verbs

        api_groups: list[str] | Unset = UNSET
        if not isinstance(self.api_groups, Unset):
            api_groups = self.api_groups

        non_resource_ur_ls: list[str] | Unset = UNSET
        if not isinstance(self.non_resource_ur_ls, Unset):
            non_resource_ur_ls = self.non_resource_ur_ls

        resource_names: list[str] | Unset = UNSET
        if not isinstance(self.resource_names, Unset):
            resource_names = self.resource_names

        resources: list[str] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "verbs": verbs,
            }
        )
        if api_groups is not UNSET:
            field_dict["apiGroups"] = api_groups
        if non_resource_ur_ls is not UNSET:
            field_dict["nonResourceURLs"] = non_resource_ur_ls
        if resource_names is not UNSET:
            field_dict["resourceNames"] = resource_names
        if resources is not UNSET:
            field_dict["resources"] = resources

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        verbs = cast(list[str], d.pop("verbs"))

        api_groups = cast(list[str], d.pop("apiGroups", UNSET))

        non_resource_ur_ls = cast(list[str], d.pop("nonResourceURLs", UNSET))

        resource_names = cast(list[str], d.pop("resourceNames", UNSET))

        resources = cast(list[str], d.pop("resources", UNSET))

        rbacv_1_policy_rule = cls(
            verbs=verbs,
            api_groups=api_groups,
            non_resource_ur_ls=non_resource_ur_ls,
            resource_names=resource_names,
            resources=resources,
        )

        rbacv_1_policy_rule.additional_properties = d
        return rbacv_1_policy_rule

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
