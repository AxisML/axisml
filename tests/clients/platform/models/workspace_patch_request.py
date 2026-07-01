from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workspace_lifecycle import WorkspaceLifecycle


T = TypeVar("T", bound="WorkspacePatchRequest")


@_attrs_define
class WorkspacePatchRequest:
    """
    Example:
        {'description': 'Updated description.', 'displayName': 'Vision team dev environment (v2)', 'lifecycle':
            {'idleTimeoutSeconds': 7200}}

    Attributes:
        description (str | Unset): Updated free-text workspace description.
        display_name (str | Unset): Updated human-readable workspace label.
        lifecycle (WorkspaceLifecycle | Unset):  Example: {'idleTimeoutSeconds': 3600}.
    """

    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    lifecycle: WorkspaceLifecycle | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        description = self.description

        display_name = self.display_name

        lifecycle: dict[str, Any] | Unset = UNSET
        if not isinstance(self.lifecycle, Unset):
            lifecycle = self.lifecycle.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if lifecycle is not UNSET:
            field_dict["lifecycle"] = lifecycle

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workspace_lifecycle import WorkspaceLifecycle

        d = dict(src_dict)
        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _lifecycle = d.pop("lifecycle", UNSET)
        lifecycle: WorkspaceLifecycle | Unset
        if isinstance(_lifecycle, Unset):
            lifecycle = UNSET
        else:
            lifecycle = WorkspaceLifecycle.from_dict(_lifecycle)

        workspace_patch_request = cls(
            description=description,
            display_name=display_name,
            lifecycle=lifecycle,
        )

        workspace_patch_request.additional_properties = d
        return workspace_patch_request

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
