from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceTool")


@_attrs_define
class WorkspaceTool:
    """
    Example:
        {'label': 'JupyterLab', 'name': 'jupyter', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/lab'}

    Attributes:
        name (str): Tool identifier (jupyter, vscode, terminal, ...).
        url (str): Launch URL for the tool.
        label (str | Unset): Human-readable tool label.
    """

    name: str
    url: str
    label: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        url = self.url

        label = self.label

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "url": url,
            }
        )
        if label is not UNSET:
            field_dict["label"] = label

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        url = d.pop("url")

        label = d.pop("label", UNSET)

        workspace_tool = cls(
            name=name,
            url=url,
            label=label,
        )

        workspace_tool.additional_properties = d
        return workspace_tool

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
