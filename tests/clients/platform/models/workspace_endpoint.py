from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceEndpoint")


@_attrs_define
class WorkspaceEndpoint:
    """
    Example:
        {'accessUrl': 'https://axisml.example.com/ws/team-vision/notebook-dev/', 'internalDns': 'notebook-dev.axisml-
            team-vision.svc.cluster.local'}

    Attributes:
        access_url (str | Unset): External URL for reaching the workspace UI.
        internal_dns (str | Unset): In-cluster DNS name for the workspace service.
    """

    access_url: str | Unset = UNSET
    internal_dns: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        access_url = self.access_url

        internal_dns = self.internal_dns

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if access_url is not UNSET:
            field_dict["accessUrl"] = access_url
        if internal_dns is not UNSET:
            field_dict["internalDns"] = internal_dns

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        access_url = d.pop("accessUrl", UNSET)

        internal_dns = d.pop("internalDns", UNSET)

        workspace_endpoint = cls(
            access_url=access_url,
            internal_dns=internal_dns,
        )

        workspace_endpoint.additional_properties = d
        return workspace_endpoint

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
