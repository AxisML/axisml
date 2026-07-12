from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.rbacv_1_policy_rule import Rbacv1PolicyRule
    from ..models.tenantv_1_alpha_1rbac_role_ref import Tenantv1Alpha1RBACRoleRef


T = TypeVar("T", bound="Tenantv1Alpha1RBACSpec")


@_attrs_define
class Tenantv1Alpha1RBACSpec:
    """
    Attributes:
        role_ref (None | Tenantv1Alpha1RBACRoleRef | Unset):
        rules (list[Rbacv1PolicyRule] | Unset):
    """

    role_ref: None | Tenantv1Alpha1RBACRoleRef | Unset = UNSET
    rules: list[Rbacv1PolicyRule] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.tenantv_1_alpha_1rbac_role_ref import Tenantv1Alpha1RBACRoleRef

        role_ref: dict[str, Any] | None | Unset
        if isinstance(self.role_ref, Unset):
            role_ref = UNSET
        elif isinstance(self.role_ref, Tenantv1Alpha1RBACRoleRef):
            role_ref = self.role_ref.to_dict()
        else:
            role_ref = self.role_ref

        rules: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.rules, Unset):
            rules = []
            for rules_item_data in self.rules:
                rules_item = rules_item_data.to_dict()
                rules.append(rules_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if role_ref is not UNSET:
            field_dict["roleRef"] = role_ref
        if rules is not UNSET:
            field_dict["rules"] = rules

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.rbacv_1_policy_rule import Rbacv1PolicyRule
        from ..models.tenantv_1_alpha_1rbac_role_ref import Tenantv1Alpha1RBACRoleRef

        d = dict(src_dict)

        def _parse_role_ref(data: object) -> None | Tenantv1Alpha1RBACRoleRef | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                role_ref_type_1 = Tenantv1Alpha1RBACRoleRef.from_dict(data)

                return role_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Tenantv1Alpha1RBACRoleRef | Unset, data)

        role_ref = _parse_role_ref(d.pop("roleRef", UNSET))

        _rules = d.pop("rules", UNSET)
        rules: list[Rbacv1PolicyRule] | Unset = UNSET
        if _rules is not UNSET:
            rules = []
            for rules_item_data in _rules:
                rules_item = Rbacv1PolicyRule.from_dict(rules_item_data)

                rules.append(rules_item)

        tenantv_1_alpha_1rbac_spec = cls(
            role_ref=role_ref,
            rules=rules,
        )

        tenantv_1_alpha_1rbac_spec.additional_properties = d
        return tenantv_1_alpha_1rbac_spec

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
