from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_traffic_policy_endpoint_auth import MLTrafficPolicyEndpointAuth


T = TypeVar("T", bound="MLTrafficPolicyEndpoint")


@_attrs_define
class MLTrafficPolicyEndpoint:
    """
    Attributes:
        auth (MLTrafficPolicyEndpointAuth | None | Unset):
        hostname (str | Unset):
        path (str | Unset):
    """

    auth: MLTrafficPolicyEndpointAuth | None | Unset = UNSET
    hostname: str | Unset = UNSET
    path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_traffic_policy_endpoint_auth import MLTrafficPolicyEndpointAuth

        auth: dict[str, Any] | None | Unset
        if isinstance(self.auth, Unset):
            auth = UNSET
        elif isinstance(self.auth, MLTrafficPolicyEndpointAuth):
            auth = self.auth.to_dict()
        else:
            auth = self.auth

        hostname = self.hostname

        path = self.path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if auth is not UNSET:
            field_dict["auth"] = auth
        if hostname is not UNSET:
            field_dict["hostname"] = hostname
        if path is not UNSET:
            field_dict["path"] = path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_traffic_policy_endpoint_auth import MLTrafficPolicyEndpointAuth

        d = dict(src_dict)

        def _parse_auth(data: object) -> MLTrafficPolicyEndpointAuth | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                auth_type_1 = MLTrafficPolicyEndpointAuth.from_dict(data)

                return auth_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLTrafficPolicyEndpointAuth | None | Unset, data)

        auth = _parse_auth(d.pop("auth", UNSET))

        hostname = d.pop("hostname", UNSET)

        path = d.pop("path", UNSET)

        ml_traffic_policy_endpoint = cls(
            auth=auth,
            hostname=hostname,
            path=path,
        )

        ml_traffic_policy_endpoint.additional_properties = d
        return ml_traffic_policy_endpoint

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
