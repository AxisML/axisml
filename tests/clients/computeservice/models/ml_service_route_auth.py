from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_route_auth_api_key_config import (
        MLServiceRouteAuthAPIKeyConfig,
    )
    from ..models.ml_service_route_auth_jwt_config import MLServiceRouteAuthJWTConfig


T = TypeVar("T", bound="MLServiceRouteAuth")


@_attrs_define
class MLServiceRouteAuth:
    """
    Attributes:
        api_key (MLServiceRouteAuthAPIKeyConfig | None | Unset):
        jwt (MLServiceRouteAuthJWTConfig | None | Unset):
        type_ (str | Unset):
    """

    api_key: MLServiceRouteAuthAPIKeyConfig | None | Unset = UNSET
    jwt: MLServiceRouteAuthJWTConfig | None | Unset = UNSET
    type_: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_service_route_auth_api_key_config import (
            MLServiceRouteAuthAPIKeyConfig,
        )
        from ..models.ml_service_route_auth_jwt_config import (
            MLServiceRouteAuthJWTConfig,
        )

        api_key: dict[str, Any] | None | Unset
        if isinstance(self.api_key, Unset):
            api_key = UNSET
        elif isinstance(self.api_key, MLServiceRouteAuthAPIKeyConfig):
            api_key = self.api_key.to_dict()
        else:
            api_key = self.api_key

        jwt: dict[str, Any] | None | Unset
        if isinstance(self.jwt, Unset):
            jwt = UNSET
        elif isinstance(self.jwt, MLServiceRouteAuthJWTConfig):
            jwt = self.jwt.to_dict()
        else:
            jwt = self.jwt

        type_ = self.type_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if api_key is not UNSET:
            field_dict["apiKey"] = api_key
        if jwt is not UNSET:
            field_dict["jwt"] = jwt
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_route_auth_api_key_config import (
            MLServiceRouteAuthAPIKeyConfig,
        )
        from ..models.ml_service_route_auth_jwt_config import (
            MLServiceRouteAuthJWTConfig,
        )

        d = dict(src_dict)

        def _parse_api_key(
            data: object,
        ) -> MLServiceRouteAuthAPIKeyConfig | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                api_key_type_1 = MLServiceRouteAuthAPIKeyConfig.from_dict(data)

                return api_key_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRouteAuthAPIKeyConfig | None | Unset, data)

        api_key = _parse_api_key(d.pop("apiKey", UNSET))

        def _parse_jwt(data: object) -> MLServiceRouteAuthJWTConfig | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                jwt_type_1 = MLServiceRouteAuthJWTConfig.from_dict(data)

                return jwt_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRouteAuthJWTConfig | None | Unset, data)

        jwt = _parse_jwt(d.pop("jwt", UNSET))

        type_ = d.pop("type", UNSET)

        ml_service_route_auth = cls(
            api_key=api_key,
            jwt=jwt,
            type_=type_,
        )

        ml_service_route_auth.additional_properties = d
        return ml_service_route_auth

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
