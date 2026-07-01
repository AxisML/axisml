from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkspaceDeleteRequest")


@_attrs_define
class WorkspaceDeleteRequest:
    """
    Example:
        {'deletePvc': False}

    Attributes:
        delete_pvc (bool | Unset): When true, also delete the workspace's persistent volumes.
    """

    delete_pvc: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        delete_pvc = self.delete_pvc

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if delete_pvc is not UNSET:
            field_dict["deletePvc"] = delete_pvc

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        delete_pvc = d.pop("deletePvc", UNSET)

        workspace_delete_request = cls(
            delete_pvc=delete_pvc,
        )

        workspace_delete_request.additional_properties = d
        return workspace_delete_request

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
