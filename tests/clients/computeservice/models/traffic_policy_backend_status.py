from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="TrafficPolicyBackendStatus")


@_attrs_define
class TrafficPolicyBackendStatus:
    """
    Attributes:
        ready (bool): True when the member service is ready to receive traffic.
        service_name (str): Member MLService name.
        weight (int): Effective traffic weight currently routed to the member.
    """

    ready: bool
    service_name: str
    weight: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        ready = self.ready

        service_name = self.service_name

        weight = self.weight

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "ready": ready,
                "serviceName": service_name,
                "weight": weight,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        ready = d.pop("ready")

        service_name = d.pop("serviceName")

        weight = d.pop("weight")

        traffic_policy_backend_status = cls(
            ready=ready,
            service_name=service_name,
            weight=weight,
        )

        traffic_policy_backend_status.additional_properties = d
        return traffic_policy_backend_status

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
