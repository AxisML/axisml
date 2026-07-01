from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.problem_field_error import ProblemFieldError


T = TypeVar("T", bound="Problem")


@_attrs_define
class Problem:
    """
    Example:
        {'code': 'JOB_NOT_FOUND', 'detail': 'Job "resnet-train" was not found in tenant "team-vision".', 'instance':
            '/api/v1/jobs/resnet-train', 'status': 404, 'title': 'Not Found', 'type': 'https://docs.axisml.io/errors/not-
            found'}

    Attributes:
        code (str | Unset): Stable machine-readable error code.
        detail (str | Unset): Human-readable explanation specific to this occurrence.
        errors (list[ProblemFieldError] | Unset): Field-level validation errors, when applicable.
        instance (str | Unset): URI reference identifying the specific occurrence of the problem.
        status (int | Unset): HTTP status code for this occurrence of the problem.
        title (str | Unset): Short, human-readable summary of the problem type.
        type_ (str | Unset): URI reference identifying the problem type.
    """

    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    errors: list[ProblemFieldError] | Unset = UNSET
    instance: str | Unset = UNSET
    status: int | Unset = UNSET
    title: str | Unset = UNSET
    type_: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        code = self.code

        detail = self.detail

        errors: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.errors, Unset):
            errors = []
            for errors_item_data in self.errors:
                errors_item = errors_item_data.to_dict()
                errors.append(errors_item)

        instance = self.instance

        status = self.status

        title = self.title

        type_ = self.type_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if errors is not UNSET:
            field_dict["errors"] = errors
        if instance is not UNSET:
            field_dict["instance"] = instance
        if status is not UNSET:
            field_dict["status"] = status
        if title is not UNSET:
            field_dict["title"] = title
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.problem_field_error import ProblemFieldError

        d = dict(src_dict)
        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        _errors = d.pop("errors", UNSET)
        errors: list[ProblemFieldError] | Unset = UNSET
        if _errors is not UNSET:
            errors = []
            for errors_item_data in _errors:
                errors_item = ProblemFieldError.from_dict(errors_item_data)

                errors.append(errors_item)

        instance = d.pop("instance", UNSET)

        status = d.pop("status", UNSET)

        title = d.pop("title", UNSET)

        type_ = d.pop("type", UNSET)

        problem = cls(
            code=code,
            detail=detail,
            errors=errors,
            instance=instance,
            status=status,
            title=title,
            type_=type_,
        )

        problem.additional_properties = d
        return problem

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
