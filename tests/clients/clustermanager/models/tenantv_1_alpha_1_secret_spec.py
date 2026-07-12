from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenantv_1_alpha_1_source_secret_ref import (
        Tenantv1Alpha1SourceSecretRef,
    )


T = TypeVar("T", bound="Tenantv1Alpha1SecretSpec")


@_attrs_define
class Tenantv1Alpha1SecretSpec:
    """
    Attributes:
        name (str):
        source_secret_ref (Tenantv1Alpha1SourceSecretRef):
        type_ (str | Unset):
    """

    name: str
    source_secret_ref: Tenantv1Alpha1SourceSecretRef
    type_: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        source_secret_ref = self.source_secret_ref.to_dict()

        type_ = self.type_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "sourceSecretRef": source_secret_ref,
            }
        )
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenantv_1_alpha_1_source_secret_ref import (
            Tenantv1Alpha1SourceSecretRef,
        )

        d = dict(src_dict)
        name = d.pop("name")

        source_secret_ref = Tenantv1Alpha1SourceSecretRef.from_dict(
            d.pop("sourceSecretRef")
        )

        type_ = d.pop("type", UNSET)

        tenantv_1_alpha_1_secret_spec = cls(
            name=name,
            source_secret_ref=source_secret_ref,
            type_=type_,
        )

        tenantv_1_alpha_1_secret_spec.additional_properties = d
        return tenantv_1_alpha_1_secret_spec

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
