from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceLifecycle")


@_attrs_define
class WorkspaceLifecycle:
    """
    Example:
        {'idleTimeoutSeconds': 3600}

    Attributes:
        idle_timeout_seconds (int | Unset): Idle duration (seconds) after which the workspace is auto-stopped; 0
            disables auto-stop.
    """

    idle_timeout_seconds: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        idle_timeout_seconds = self.idle_timeout_seconds

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if idle_timeout_seconds is not UNSET:
            field_dict["idleTimeoutSeconds"] = idle_timeout_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        idle_timeout_seconds = d.pop("idleTimeoutSeconds", UNSET)

        workspace_lifecycle = cls(
            idle_timeout_seconds=idle_timeout_seconds,
        )

        workspace_lifecycle.additional_properties = d
        return workspace_lifecycle

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
