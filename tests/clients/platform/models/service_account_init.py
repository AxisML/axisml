from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.service_account_init_rbac import ServiceAccountInitRbac


T = TypeVar("T", bound="ServiceAccountInit")


@_attrs_define
class ServiceAccountInit:
    """
    Example:
        {'imagePullSecrets': ['registry-pull'], 'name': 'trainer', 'rbac': {'roleRefs': [{'kind': 'ClusterRole', 'name':
            'edit'}]}}

    Attributes:
        name (str): Name of the ServiceAccount to create in the tenant namespace.
        image_pull_secrets (list[str] | Unset): Image pull Secret names attached to the ServiceAccount.
        rbac (ServiceAccountInitRbac | Unset): RBAC rules to bind to the ServiceAccount.
    """

    name: str
    image_pull_secrets: list[str] | Unset = UNSET
    rbac: ServiceAccountInitRbac | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        image_pull_secrets: list[str] | Unset = UNSET
        if not isinstance(self.image_pull_secrets, Unset):
            image_pull_secrets = self.image_pull_secrets

        rbac: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rbac, Unset):
            rbac = self.rbac.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if image_pull_secrets is not UNSET:
            field_dict["imagePullSecrets"] = image_pull_secrets
        if rbac is not UNSET:
            field_dict["rbac"] = rbac

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.service_account_init_rbac import ServiceAccountInitRbac

        d = dict(src_dict)
        name = d.pop("name")

        image_pull_secrets = cast(list[str], d.pop("imagePullSecrets", UNSET))

        _rbac = d.pop("rbac", UNSET)
        rbac: ServiceAccountInitRbac | Unset
        if isinstance(_rbac, Unset):
            rbac = UNSET
        else:
            rbac = ServiceAccountInitRbac.from_dict(_rbac)

        service_account_init = cls(
            name=name,
            image_pull_secrets=image_pull_secrets,
            rbac=rbac,
        )

        service_account_init.additional_properties = d
        return service_account_init

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
