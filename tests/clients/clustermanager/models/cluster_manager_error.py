from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ClusterManagerError")


@_attrs_define
class ClusterManagerError:
    """
    Example:
        {'code': 'NotFound', 'detail': 'ResourcePool "gpu-a100" not found.', 'status': 404, 'title': 'Not found',
            'type': 'about:blank'}

    Attributes:
        status (int): HTTP status code for this occurrence of the problem.
        title (str): Short, human-readable summary of the problem.
        type_ (str): URI reference identifying the problem type (RFC 7807); about:blank when unspecified.
        code (str | Unset): Stable machine-readable error code for programmatic handling.
        detail (str | Unset): Human-readable explanation specific to this occurrence.
    """

    status: int
    title: str
    type_: str
    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status = self.status

        title = self.title

        type_ = self.type_

        code = self.code

        detail = self.detail

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "status": status,
                "title": title,
                "type": type_,
            }
        )
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = d.pop("status")

        title = d.pop("title")

        type_ = d.pop("type")

        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        cluster_manager_error = cls(
            status=status,
            title=title,
            type_=type_,
            code=code,
            detail=detail,
        )

        cluster_manager_error.additional_properties = d
        return cluster_manager_error

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
