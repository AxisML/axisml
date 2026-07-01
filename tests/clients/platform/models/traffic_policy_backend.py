from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.traffic_policy_backend_role import TrafficPolicyBackendRole
from ..types import UNSET, Unset

T = TypeVar("T", bound="TrafficPolicyBackend")


@_attrs_define
class TrafficPolicyBackend:
    """
    Example:
        {'actualPct': 90, 'ready': True, 'role': 'stable', 'serviceName': 'resnet-serving-v1', 'weight': 90}

    Attributes:
        service_name (str): Name of the member online service receiving a share of traffic.
        weight (int): Configured traffic weight (0-100); weights across backends sum to 100.
        actual_pct (int | Unset): Live percentage of traffic actually routed to this backend (read-only).
        ready (bool | Unset): Whether the backend service is ready to serve (read-only).
        role (TrafficPolicyBackendRole | Unset):
    """

    service_name: str
    weight: int
    actual_pct: int | Unset = UNSET
    ready: bool | Unset = UNSET
    role: TrafficPolicyBackendRole | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        service_name = self.service_name

        weight = self.weight

        actual_pct = self.actual_pct

        ready = self.ready

        role: str | Unset = UNSET
        if not isinstance(self.role, Unset):
            role = self.role.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "serviceName": service_name,
                "weight": weight,
            }
        )
        if actual_pct is not UNSET:
            field_dict["actualPct"] = actual_pct
        if ready is not UNSET:
            field_dict["ready"] = ready
        if role is not UNSET:
            field_dict["role"] = role

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        service_name = d.pop("serviceName")

        weight = d.pop("weight")

        actual_pct = d.pop("actualPct", UNSET)

        ready = d.pop("ready", UNSET)

        _role = d.pop("role", UNSET)
        role: TrafficPolicyBackendRole | Unset
        if isinstance(_role, Unset):
            role = UNSET
        else:
            role = TrafficPolicyBackendRole(_role)

        traffic_policy_backend = cls(
            service_name=service_name,
            weight=weight,
            actual_pct=actual_pct,
            ready=ready,
            role=role,
        )

        traffic_policy_backend.additional_properties = d
        return traffic_policy_backend

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
