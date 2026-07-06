from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceImage")


@_attrs_define
class WorkspaceImage:
    """
    Example:
        {'defaultPort': 8888, 'description': 'JupyterLab data-science environment (public).', 'displayName':
            'JupyterLab', 'kind': 'jupyter', 'public': True, 'ref': 'registry.axisml.io/dev/jupyter:3.0.0'}

    Attributes:
        display_name (str): Human-readable image label.
        ref (str): Container image reference.
        default_port (int | Unset): Default container port the dev server listens on.
        description (str | Unset): Short description of the environment.
        kind (str | Unset): Environment kind (jupyter, vscode, terminal, custom).
        public (bool | Unset): Whether the image is a public/shared base image.
    """

    display_name: str
    ref: str
    default_port: int | Unset = UNSET
    description: str | Unset = UNSET
    kind: str | Unset = UNSET
    public: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        display_name = self.display_name

        ref = self.ref

        default_port = self.default_port

        description = self.description

        kind = self.kind

        public = self.public

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "displayName": display_name,
                "ref": ref,
            }
        )
        if default_port is not UNSET:
            field_dict["defaultPort"] = default_port
        if description is not UNSET:
            field_dict["description"] = description
        if kind is not UNSET:
            field_dict["kind"] = kind
        if public is not UNSET:
            field_dict["public"] = public

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        display_name = d.pop("displayName")

        ref = d.pop("ref")

        default_port = d.pop("defaultPort", UNSET)

        description = d.pop("description", UNSET)

        kind = d.pop("kind", UNSET)

        public = d.pop("public", UNSET)

        workspace_image = cls(
            display_name=display_name,
            ref=ref,
            default_port=default_port,
            description=description,
            kind=kind,
            public=public,
        )

        workspace_image.additional_properties = d
        return workspace_image

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
