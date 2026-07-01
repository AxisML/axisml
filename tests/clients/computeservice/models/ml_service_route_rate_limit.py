from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLServiceRouteRateLimit")


@_attrs_define
class MLServiceRouteRateLimit:
    """
    Attributes:
        requests_per_second (int):
        burst (int | Unset):
    """

    requests_per_second: int
    burst: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        requests_per_second = self.requests_per_second

        burst = self.burst

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "requestsPerSecond": requests_per_second,
            }
        )
        if burst is not UNSET:
            field_dict["burst"] = burst

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        requests_per_second = d.pop("requestsPerSecond")

        burst = d.pop("burst", UNSET)

        ml_service_route_rate_limit = cls(
            requests_per_second=requests_per_second,
            burst=burst,
        )

        ml_service_route_rate_limit.additional_properties = d
        return ml_service_route_rate_limit

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
