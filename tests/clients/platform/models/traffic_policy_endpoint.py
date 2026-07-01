from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="TrafficPolicyEndpoint")


@_attrs_define
class TrafficPolicyEndpoint:
    """
    Example:
        {'hostname': 'infer.axisml.io', 'path': '/services/team-vision/resnet-serving/'}

    Attributes:
        hostname (str | Unset): Optional external hostname for the entry. Immutable after creation.
        path (str | Unset): External URL path; empty auto-generates /services/<tenant>/<name>/. Immutable after
            creation.
    """

    hostname: str | Unset = UNSET
    path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        hostname = self.hostname

        path = self.path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if hostname is not UNSET:
            field_dict["hostname"] = hostname
        if path is not UNSET:
            field_dict["path"] = path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hostname = d.pop("hostname", UNSET)

        path = d.pop("path", UNSET)

        traffic_policy_endpoint = cls(
            hostname=hostname,
            path=path,
        )

        traffic_policy_endpoint.additional_properties = d
        return traffic_policy_endpoint

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
