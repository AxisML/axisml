from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_route_auth import MLServiceRouteAuth
    from ..models.ml_service_route_rate_limit import MLServiceRouteRateLimit


T = TypeVar("T", bound="MLServiceRoute")


@_attrs_define
class MLServiceRoute:
    """
    Attributes:
        enabled (bool):
        auth (MLServiceRouteAuth | None | Unset):
        hostname (str | Unset):
        path (str | Unset):
        port_name (str | Unset):
        rate_limit (MLServiceRouteRateLimit | None | Unset):
        target_role (str | Unset):
        timeout (str | Unset):
    """

    enabled: bool
    auth: MLServiceRouteAuth | None | Unset = UNSET
    hostname: str | Unset = UNSET
    path: str | Unset = UNSET
    port_name: str | Unset = UNSET
    rate_limit: MLServiceRouteRateLimit | None | Unset = UNSET
    target_role: str | Unset = UNSET
    timeout: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_service_route_auth import MLServiceRouteAuth
        from ..models.ml_service_route_rate_limit import MLServiceRouteRateLimit

        enabled = self.enabled

        auth: dict[str, Any] | None | Unset
        if isinstance(self.auth, Unset):
            auth = UNSET
        elif isinstance(self.auth, MLServiceRouteAuth):
            auth = self.auth.to_dict()
        else:
            auth = self.auth

        hostname = self.hostname

        path = self.path

        port_name = self.port_name

        rate_limit: dict[str, Any] | None | Unset
        if isinstance(self.rate_limit, Unset):
            rate_limit = UNSET
        elif isinstance(self.rate_limit, MLServiceRouteRateLimit):
            rate_limit = self.rate_limit.to_dict()
        else:
            rate_limit = self.rate_limit

        target_role = self.target_role

        timeout = self.timeout

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "enabled": enabled,
            }
        )
        if auth is not UNSET:
            field_dict["auth"] = auth
        if hostname is not UNSET:
            field_dict["hostname"] = hostname
        if path is not UNSET:
            field_dict["path"] = path
        if port_name is not UNSET:
            field_dict["portName"] = port_name
        if rate_limit is not UNSET:
            field_dict["rateLimit"] = rate_limit
        if target_role is not UNSET:
            field_dict["targetRole"] = target_role
        if timeout is not UNSET:
            field_dict["timeout"] = timeout

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_route_auth import MLServiceRouteAuth
        from ..models.ml_service_route_rate_limit import MLServiceRouteRateLimit

        d = dict(src_dict)
        enabled = d.pop("enabled")

        def _parse_auth(data: object) -> MLServiceRouteAuth | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                auth_type_1 = MLServiceRouteAuth.from_dict(data)

                return auth_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRouteAuth | None | Unset, data)

        auth = _parse_auth(d.pop("auth", UNSET))

        hostname = d.pop("hostname", UNSET)

        path = d.pop("path", UNSET)

        port_name = d.pop("portName", UNSET)

        def _parse_rate_limit(data: object) -> MLServiceRouteRateLimit | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                rate_limit_type_1 = MLServiceRouteRateLimit.from_dict(data)

                return rate_limit_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRouteRateLimit | None | Unset, data)

        rate_limit = _parse_rate_limit(d.pop("rateLimit", UNSET))

        target_role = d.pop("targetRole", UNSET)

        timeout = d.pop("timeout", UNSET)

        ml_service_route = cls(
            enabled=enabled,
            auth=auth,
            hostname=hostname,
            path=path,
            port_name=port_name,
            rate_limit=rate_limit,
            target_role=target_role,
            timeout=timeout,
        )

        ml_service_route.additional_properties = d
        return ml_service_route

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
