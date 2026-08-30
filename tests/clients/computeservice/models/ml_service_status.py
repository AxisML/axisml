from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLServiceStatus")


@_attrs_define
class MLServiceStatus:
    """
    Attributes:
        admitted_replicas (int): Primary-role replicas holding a durable capacity and quota reservation.
        ready_replicas (int): Number of replicas that have passed readiness.
        admission_message (str | Unset): Human-readable detail for the current admission wait.
        admission_reason (str | Unset): Stable reason why desired service replicas are still waiting for admission:
            InventoryUnavailable, QuotaUnavailable, QuotaExceeded, NoMatchingNode, or InsufficientResources.
        endpoint (str | Unset): Resolved external endpoint URL when a route is enabled.
        message (str | Unset): Human-readable runtime status detail for the current phase.
    """

    admitted_replicas: int
    ready_replicas: int
    admission_message: str | Unset = UNSET
    admission_reason: str | Unset = UNSET
    endpoint: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        admitted_replicas = self.admitted_replicas

        ready_replicas = self.ready_replicas

        admission_message = self.admission_message

        admission_reason = self.admission_reason

        endpoint = self.endpoint

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "admittedReplicas": admitted_replicas,
                "readyReplicas": ready_replicas,
            }
        )
        if admission_message is not UNSET:
            field_dict["admissionMessage"] = admission_message
        if admission_reason is not UNSET:
            field_dict["admissionReason"] = admission_reason
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        admitted_replicas = d.pop("admittedReplicas")

        ready_replicas = d.pop("readyReplicas")

        admission_message = d.pop("admissionMessage", UNSET)

        admission_reason = d.pop("admissionReason", UNSET)

        endpoint = d.pop("endpoint", UNSET)

        message = d.pop("message", UNSET)

        ml_service_status = cls(
            admitted_replicas=admitted_replicas,
            ready_replicas=ready_replicas,
            admission_message=admission_message,
            admission_reason=admission_reason,
            endpoint=endpoint,
            message=message,
        )

        ml_service_status.additional_properties = d
        return ml_service_status

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
