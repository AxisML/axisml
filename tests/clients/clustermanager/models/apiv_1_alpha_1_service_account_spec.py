from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.apiv_1_alpha_1rbac_spec import Apiv1Alpha1RBACSpec


T = TypeVar("T", bound="Apiv1Alpha1ServiceAccountSpec")


@_attrs_define
class Apiv1Alpha1ServiceAccountSpec:
    """
    Attributes:
        name (str):
        image_pull_secrets (list[str] | Unset):
        rbac (Apiv1Alpha1RBACSpec | None | Unset):
    """

    name: str
    image_pull_secrets: list[str] | Unset = UNSET
    rbac: Apiv1Alpha1RBACSpec | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.apiv_1_alpha_1rbac_spec import Apiv1Alpha1RBACSpec

        name = self.name

        image_pull_secrets: list[str] | Unset = UNSET
        if not isinstance(self.image_pull_secrets, Unset):
            image_pull_secrets = self.image_pull_secrets

        rbac: dict[str, Any] | None | Unset
        if isinstance(self.rbac, Unset):
            rbac = UNSET
        elif isinstance(self.rbac, Apiv1Alpha1RBACSpec):
            rbac = self.rbac.to_dict()
        else:
            rbac = self.rbac

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
        from ..models.apiv_1_alpha_1rbac_spec import Apiv1Alpha1RBACSpec

        d = dict(src_dict)
        name = d.pop("name")

        image_pull_secrets = cast(list[str], d.pop("imagePullSecrets", UNSET))

        def _parse_rbac(data: object) -> Apiv1Alpha1RBACSpec | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                rbac_type_1 = Apiv1Alpha1RBACSpec.from_dict(data)

                return rbac_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Apiv1Alpha1RBACSpec | None | Unset, data)

        rbac = _parse_rbac(d.pop("rbac", UNSET))

        apiv_1_alpha_1_service_account_spec = cls(
            name=name,
            image_pull_secrets=image_pull_secrets,
            rbac=rbac,
        )

        apiv_1_alpha_1_service_account_spec.additional_properties = d
        return apiv_1_alpha_1_service_account_spec

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
