from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.traffic_policy_backend_status import TrafficPolicyBackendStatus


T = TypeVar("T", bound="TrafficPolicyStatus")


@_attrs_define
class TrafficPolicyStatus:
    """
    Attributes:
        backends (list[TrafficPolicyBackendStatus] | Unset): Per-member effective weight and readiness.
        endpoint (str | Unset): Resolved external endpoint URL fronting the member services.
        message (str | Unset): Human-readable status detail for the current phase.
    """

    backends: list[TrafficPolicyBackendStatus] | Unset = UNSET
    endpoint: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.backends, Unset):
            backends = []
            for backends_item_data in self.backends:
                backends_item = backends_item_data.to_dict()
                backends.append(backends_item)

        endpoint = self.endpoint

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if backends is not UNSET:
            field_dict["backends"] = backends
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.traffic_policy_backend_status import TrafficPolicyBackendStatus

        d = dict(src_dict)
        _backends = d.pop("backends", UNSET)
        backends: list[TrafficPolicyBackendStatus] | Unset = UNSET
        if _backends is not UNSET:
            backends = []
            for backends_item_data in _backends:
                backends_item = TrafficPolicyBackendStatus.from_dict(backends_item_data)

                backends.append(backends_item)

        endpoint = d.pop("endpoint", UNSET)

        message = d.pop("message", UNSET)

        traffic_policy_status = cls(
            backends=backends,
            endpoint=endpoint,
            message=message,
        )

        traffic_policy_status.additional_properties = d
        return traffic_policy_status

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
