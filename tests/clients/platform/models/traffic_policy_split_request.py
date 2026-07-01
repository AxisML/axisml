from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.traffic_policy_backend_spec import TrafficPolicyBackendSpec


T = TypeVar("T", bound="TrafficPolicySplitRequest")


@_attrs_define
class TrafficPolicySplitRequest:
    """
    Example:
        {'backends': [{'role': 'stable', 'serviceName': 'resnet-serving-v1', 'weight': 75}, {'role': 'canary',
            'serviceName': 'resnet-serving-v2', 'weight': 25}], 'canaryPercent': 25}

    Attributes:
        backends (list[TrafficPolicyBackendSpec] | Unset): Updated backend weights for weighted mode (weights sum to
            100).
        canary_percent (int | None | Unset): Updated canary traffic percent for canary mode (stable = 100−p).
    """

    backends: list[TrafficPolicyBackendSpec] | Unset = UNSET
    canary_percent: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.backends, Unset):
            backends = []
            for backends_item_data in self.backends:
                backends_item = backends_item_data.to_dict()
                backends.append(backends_item)

        canary_percent: int | None | Unset
        if isinstance(self.canary_percent, Unset):
            canary_percent = UNSET
        else:
            canary_percent = self.canary_percent

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if backends is not UNSET:
            field_dict["backends"] = backends
        if canary_percent is not UNSET:
            field_dict["canaryPercent"] = canary_percent

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.traffic_policy_backend_spec import TrafficPolicyBackendSpec

        d = dict(src_dict)
        _backends = d.pop("backends", UNSET)
        backends: list[TrafficPolicyBackendSpec] | Unset = UNSET
        if _backends is not UNSET:
            backends = []
            for backends_item_data in _backends:
                backends_item = TrafficPolicyBackendSpec.from_dict(backends_item_data)

                backends.append(backends_item)

        def _parse_canary_percent(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        canary_percent = _parse_canary_percent(d.pop("canaryPercent", UNSET))

        traffic_policy_split_request = cls(
            backends=backends,
            canary_percent=canary_percent,
        )

        traffic_policy_split_request.additional_properties = d
        return traffic_policy_split_request

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
