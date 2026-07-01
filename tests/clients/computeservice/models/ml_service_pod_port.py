from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLServicePodPort")


@_attrs_define
class MLServicePodPort:
    """
    Attributes:
        container_port (int):
        name (str):
        protocol (str | Unset):
    """

    container_port: int
    name: str
    protocol: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        container_port = self.container_port

        name = self.name

        protocol = self.protocol

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "containerPort": container_port,
                "name": name,
            }
        )
        if protocol is not UNSET:
            field_dict["protocol"] = protocol

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        container_port = d.pop("containerPort")

        name = d.pop("name")

        protocol = d.pop("protocol", UNSET)

        ml_service_pod_port = cls(
            container_port=container_port,
            name=name,
            protocol=protocol,
        )

        ml_service_pod_port.additional_properties = d
        return ml_service_pod_port

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
