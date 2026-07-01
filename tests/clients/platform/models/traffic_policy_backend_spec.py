from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.traffic_policy_backend_role import TrafficPolicyBackendRole
from ..types import UNSET, Unset

T = TypeVar("T", bound="TrafficPolicyBackendSpec")


@_attrs_define
class TrafficPolicyBackendSpec:
    """
    Example:
        {'role': 'canary', 'serviceName': 'resnet-serving-v2', 'weight': 10}

    Attributes:
        service_name (str): Name of the member online service to route traffic to.
        role (TrafficPolicyBackendRole | Unset):
        weight (int | Unset): Traffic weight (0-100); weights across backends sum to 100 for weighted mode.
    """

    service_name: str
    role: TrafficPolicyBackendRole | Unset = UNSET
    weight: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        service_name = self.service_name

        role: str | Unset = UNSET
        if not isinstance(self.role, Unset):
            role = self.role.value

        weight = self.weight

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "serviceName": service_name,
            }
        )
        if role is not UNSET:
            field_dict["role"] = role
        if weight is not UNSET:
            field_dict["weight"] = weight

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        service_name = d.pop("serviceName")

        _role = d.pop("role", UNSET)
        role: TrafficPolicyBackendRole | Unset
        if isinstance(_role, Unset):
            role = UNSET
        else:
            role = TrafficPolicyBackendRole(_role)

        weight = d.pop("weight", UNSET)

        traffic_policy_backend_spec = cls(
            service_name=service_name,
            role=role,
            weight=weight,
        )

        traffic_policy_backend_spec.additional_properties = d
        return traffic_policy_backend_spec

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
