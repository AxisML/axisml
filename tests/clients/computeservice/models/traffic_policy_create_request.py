from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_traffic_policy_backend_member import MLTrafficPolicyBackendMember
    from ..models.ml_traffic_policy_endpoint import MLTrafficPolicyEndpoint
    from ..models.traffic_policy_create_request_annotations import (
        TrafficPolicyCreateRequestAnnotations,
    )
    from ..models.traffic_policy_create_request_labels import (
        TrafficPolicyCreateRequestLabels,
    )


T = TypeVar("T", bound="TrafficPolicyCreateRequest")


@_attrs_define
class TrafficPolicyCreateRequest:
    """
    Example:
        {'backends': [{'role': 'stable', 'serviceName': 'llama3-8b', 'weight': 90}, {'role': 'canary', 'serviceName':
            'llama3-8b-v2', 'weight': 10}], 'description': 'Canary 10% traffic to v2.', 'displayName': 'Llama-3 canary
            release', 'endpoint': {'auth': {'jwt': {'audience': 'axisml-inference', 'issuer': 'https://auth.axisml.io',
            'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'}, 'type': 'jwt'}, 'hostname': 'llama3-8b.team-
            vision.axisml.io', 'path': '/v1'}, 'labels': {'team': 'vision'}, 'mode': 'canary', 'name': 'llama3-canary'}

    Attributes:
        backends (list[MLTrafficPolicyBackendMember]): Member MLServices and their weights (at least one).
        mode (str): Traffic split mode (weighted, canary, bluegreen); immutable after create.
        name (str): Traffic policy name, unique within the namespace.
        annotations (TrafficPolicyCreateRequestAnnotations | Unset): User-defined annotations stored on the row and
            stamped onto the CR.
        description (str | Unset): Free-text policy description.
        display_name (str | Unset): Human-readable policy label.
        endpoint (MLTrafficPolicyEndpoint | Unset):
        labels (TrafficPolicyCreateRequestLabels | Unset): User-defined labels stored on the row and stamped onto the
            CR.
    """

    backends: list[MLTrafficPolicyBackendMember]
    mode: str
    name: str
    annotations: TrafficPolicyCreateRequestAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    endpoint: MLTrafficPolicyEndpoint | Unset = UNSET
    labels: TrafficPolicyCreateRequestLabels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends = []
        for backends_item_data in self.backends:
            backends_item = backends_item_data.to_dict()
            backends.append(backends_item)

        mode = self.mode

        name = self.name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        endpoint: dict[str, Any] | Unset = UNSET
        if not isinstance(self.endpoint, Unset):
            endpoint = self.endpoint.to_dict()

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backends": backends,
                "mode": mode,
                "name": name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if labels is not UNSET:
            field_dict["labels"] = labels

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_traffic_policy_backend_member import (
            MLTrafficPolicyBackendMember,
        )
        from ..models.ml_traffic_policy_endpoint import MLTrafficPolicyEndpoint
        from ..models.traffic_policy_create_request_annotations import (
            TrafficPolicyCreateRequestAnnotations,
        )
        from ..models.traffic_policy_create_request_labels import (
            TrafficPolicyCreateRequestLabels,
        )

        d = dict(src_dict)
        backends = []
        _backends = d.pop("backends")
        for backends_item_data in _backends:
            backends_item = MLTrafficPolicyBackendMember.from_dict(backends_item_data)

            backends.append(backends_item)

        mode = d.pop("mode")

        name = d.pop("name")

        _annotations = d.pop("annotations", UNSET)
        annotations: TrafficPolicyCreateRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = TrafficPolicyCreateRequestAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _endpoint = d.pop("endpoint", UNSET)
        endpoint: MLTrafficPolicyEndpoint | Unset
        if isinstance(_endpoint, Unset):
            endpoint = UNSET
        else:
            endpoint = MLTrafficPolicyEndpoint.from_dict(_endpoint)

        _labels = d.pop("labels", UNSET)
        labels: TrafficPolicyCreateRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = TrafficPolicyCreateRequestLabels.from_dict(_labels)

        traffic_policy_create_request = cls(
            backends=backends,
            mode=mode,
            name=name,
            annotations=annotations,
            description=description,
            display_name=display_name,
            endpoint=endpoint,
            labels=labels,
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
