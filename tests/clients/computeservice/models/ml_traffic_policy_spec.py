from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.ml_traffic_policy_backend import MLTrafficPolicyBackend
    from ..models.ml_traffic_policy_backend_member import MLTrafficPolicyBackendMember
    from ..models.ml_traffic_policy_endpoint import MLTrafficPolicyEndpoint


T = TypeVar("T", bound="MLTrafficPolicySpec")


@_attrs_define
class MLTrafficPolicySpec:
    """
    Attributes:
        backend (MLTrafficPolicyBackend):
        backends (list[MLTrafficPolicyBackendMember]):
        endpoint (MLTrafficPolicyEndpoint):
        mode (str):
    """

    backend: MLTrafficPolicyBackend
    backends: list[MLTrafficPolicyBackendMember]
    endpoint: MLTrafficPolicyEndpoint
    mode: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backend = self.backend.to_dict()

        backends = []
        for backends_item_data in self.backends:
            backends_item = backends_item_data.to_dict()
            backends.append(backends_item)

        endpoint = self.endpoint.to_dict()

        mode = self.mode

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backend": backend,
                "backends": backends,
                "endpoint": endpoint,
                "mode": mode,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_traffic_policy_backend import MLTrafficPolicyBackend
        from ..models.ml_traffic_policy_backend_member import (
            MLTrafficPolicyBackendMember,
        )
        from ..models.ml_traffic_policy_endpoint import MLTrafficPolicyEndpoint

        d = dict(src_dict)
        backend = MLTrafficPolicyBackend.from_dict(d.pop("backend"))

        backends = []
        _backends = d.pop("backends")
        for backends_item_data in _backends:
            backends_item = MLTrafficPolicyBackendMember.from_dict(backends_item_data)

            backends.append(backends_item)

        endpoint = MLTrafficPolicyEndpoint.from_dict(d.pop("endpoint"))

        mode = d.pop("mode")

        ml_traffic_policy_spec = cls(
            backend=backend,
            backends=backends,
            endpoint=endpoint,
            mode=mode,
        )

        ml_traffic_policy_spec.additional_properties = d
        return ml_traffic_policy_spec

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
