from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="MLServiceRouteAuthAPIKeyConfig")


@_attrs_define
class MLServiceRouteAuthAPIKeyConfig:
    """
    Attributes:
        secret_ref (Corev1LocalObjectReference):
    """

    secret_ref: Corev1LocalObjectReference
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        secret_ref = self.secret_ref.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "secretRef": secret_ref,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        secret_ref = Corev1LocalObjectReference.from_dict(d.pop("secretRef"))

        ml_service_route_auth_api_key_config = cls(
            secret_ref=secret_ref,
        )

        ml_service_route_auth_api_key_config.additional_properties = d
        return ml_service_route_auth_api_key_config

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
