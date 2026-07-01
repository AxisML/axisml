from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.traffic_policy_weight_update import TrafficPolicyWeightUpdate


T = TypeVar("T", bound="TrafficPolicySplitRequest")


@_attrs_define
class TrafficPolicySplitRequest:
    """
    Example:
        {'backends': [{'serviceName': 'llama3-8b', 'weight': 80}, {'serviceName': 'llama3-8b-v2', 'weight': 20}]}

    Attributes:
        backends (list[TrafficPolicyWeightUpdate]): Per-backend weight updates; only listed backends change.
    """

    backends: list[TrafficPolicyWeightUpdate]
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends = []
        for backends_item_data in self.backends:
            backends_item = backends_item_data.to_dict()
            backends.append(backends_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backends": backends,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.traffic_policy_weight_update import TrafficPolicyWeightUpdate

        d = dict(src_dict)
        backends = []
        _backends = d.pop("backends")
        for backends_item_data in _backends:
            backends_item = TrafficPolicyWeightUpdate.from_dict(backends_item_data)

            backends.append(backends_item)

        traffic_policy_split_request = cls(
            backends=backends,
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
