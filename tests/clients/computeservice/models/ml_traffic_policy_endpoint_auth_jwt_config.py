from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLTrafficPolicyEndpointAuthJWTConfig")


@_attrs_define
class MLTrafficPolicyEndpointAuthJWTConfig:
    """
    Attributes:
        audience (str | Unset):
        issuer (str | Unset):
        jwks_uri (str | Unset):
    """

    audience: str | Unset = UNSET
    issuer: str | Unset = UNSET
    jwks_uri: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        audience = self.audience

        issuer = self.issuer

        jwks_uri = self.jwks_uri

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if audience is not UNSET:
            field_dict["audience"] = audience
        if issuer is not UNSET:
            field_dict["issuer"] = issuer
        if jwks_uri is not UNSET:
            field_dict["jwksUri"] = jwks_uri

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audience = d.pop("audience", UNSET)

        issuer = d.pop("issuer", UNSET)

        jwks_uri = d.pop("jwksUri", UNSET)

        ml_traffic_policy_endpoint_auth_jwt_config = cls(
            audience=audience,
            issuer=issuer,
            jwks_uri=jwks_uri,
        )

        ml_traffic_policy_endpoint_auth_jwt_config.additional_properties = d
        return ml_traffic_policy_endpoint_auth_jwt_config

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
