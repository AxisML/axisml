from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.traffic_policy_mode import TrafficPolicyMode
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.traffic_policy_backend_spec import TrafficPolicyBackendSpec
    from ..models.traffic_policy_endpoint import TrafficPolicyEndpoint


T = TypeVar("T", bound="TrafficPolicyCreateRequest")


@_attrs_define
class TrafficPolicyCreateRequest:
    """
    Example:
        {'backends': [{'role': 'stable', 'serviceName': 'resnet-serving-v1', 'weight': 90}, {'role': 'canary',
            'serviceName': 'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'description': 'Canary traffic split
            for the ResNet-50 online inference service.', 'displayName': 'ResNet inference traffic', 'endpoint':
            {'hostname': 'infer.axisml.io', 'path': '/services/team-vision/resnet-serving/'}, 'mode': 'canary', 'name':
            'resnet-serving'}

    Attributes:
        backends (list[TrafficPolicyBackendSpec]): Member online services and their weights (≥1).
        mode (TrafficPolicyMode):
        name (str): Traffic policy name (unique within the tenant).
        canary_percent (int | Unset): Initial canary traffic percent for canary mode.
        description (str | Unset): Free-text policy description.
        display_name (str | Unset): Human-readable policy label.
        endpoint (TrafficPolicyEndpoint | Unset):  Example: {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}.
    """

    backends: list[TrafficPolicyBackendSpec]
    mode: TrafficPolicyMode
    name: str
    canary_percent: int | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    endpoint: TrafficPolicyEndpoint | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends = []
        for backends_item_data in self.backends:
            backends_item = backends_item_data.to_dict()
            backends.append(backends_item)

        mode = self.mode.value

        name = self.name

        canary_percent = self.canary_percent

        description = self.description

        display_name = self.display_name

        endpoint: dict[str, Any] | Unset = UNSET
        if not isinstance(self.endpoint, Unset):
            endpoint = self.endpoint.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backends": backends,
                "mode": mode,
                "name": name,
            }
        )
        if canary_percent is not UNSET:
            field_dict["canaryPercent"] = canary_percent
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.traffic_policy_backend_spec import TrafficPolicyBackendSpec
        from ..models.traffic_policy_endpoint import TrafficPolicyEndpoint

        d = dict(src_dict)
        backends = []
        _backends = d.pop("backends")
        for backends_item_data in _backends:
            backends_item = TrafficPolicyBackendSpec.from_dict(backends_item_data)

            backends.append(backends_item)

        mode = TrafficPolicyMode(d.pop("mode"))

        name = d.pop("name")

        canary_percent = d.pop("canaryPercent", UNSET)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _endpoint = d.pop("endpoint", UNSET)
        endpoint: TrafficPolicyEndpoint | Unset
        if isinstance(_endpoint, Unset):
            endpoint = UNSET
        else:
            endpoint = TrafficPolicyEndpoint.from_dict(_endpoint)

        traffic_policy_create_request = cls(
            backends=backends,
            mode=mode,
            name=name,
            canary_percent=canary_percent,
            description=description,
            display_name=display_name,
            endpoint=endpoint,
        )

        traffic_policy_create_request.additional_properties = d
        return traffic_policy_create_request

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
