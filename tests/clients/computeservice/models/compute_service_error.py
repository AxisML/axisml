from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.compute_service_error_code import ComputeServiceErrorCode
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.compute_service_error_details import ComputeServiceErrorDetails


T = TypeVar("T", bound="ComputeServiceError")


@_attrs_define
class ComputeServiceError:
    """
    Example:
        {'code': 'quota_exceeded', 'detail': 'requested 8 nvidia.com/gpu exceeds the remaining quota of 2', 'details':
            {'available': '2', 'requested': '8', 'resource': 'nvidia.com/gpu'}, 'instance': '/api/v1/namespaces/team-
            vision/mlruns', 'status': 422, 'title': 'ElasticQuota team-vision exhausted', 'type':
            'https://axisml.io/errors/quota_exceeded'}

    Attributes:
        code (ComputeServiceErrorCode): Discrete business error class for programmatic handling.
        status (int): HTTP status code for this occurrence of the problem.
        title (str): Short, human-readable summary of the problem.
        type_ (str): URI identifying the problem type (https://axisml.io/errors/<code>).
        detail (str | Unset): Human-readable explanation specific to this occurrence.
        details (ComputeServiceErrorDetails | Unset): Optional structured field-level error details.
        instance (str | Unset): URI reference identifying the specific occurrence (the request path).
    """

    code: ComputeServiceErrorCode
    status: int
    title: str
    type_: str
    detail: str | Unset = UNSET
    details: ComputeServiceErrorDetails | Unset = UNSET
    instance: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        code = self.code.value

        status = self.status

        title = self.title

        type_ = self.type_

        detail = self.detail

        details: dict[str, Any] | Unset = UNSET
        if not isinstance(self.details, Unset):
            details = self.details.to_dict()

        instance = self.instance

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "code": code,
                "status": status,
                "title": title,
                "type": type_,
            }
        )
        if detail is not UNSET:
            field_dict["detail"] = detail
        if details is not UNSET:
            field_dict["details"] = details
        if instance is not UNSET:
            field_dict["instance"] = instance

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.compute_service_error_details import ComputeServiceErrorDetails

        d = dict(src_dict)
        code = ComputeServiceErrorCode(d.pop("code"))

        status = d.pop("status")

        title = d.pop("title")

        type_ = d.pop("type")

        detail = d.pop("detail", UNSET)

        _details = d.pop("details", UNSET)
        details: ComputeServiceErrorDetails | Unset
        if isinstance(_details, Unset):
            details = UNSET
        else:
            details = ComputeServiceErrorDetails.from_dict(_details)

        instance = d.pop("instance", UNSET)

        compute_service_error = cls(
            code=code,
            status=status,
            title=title,
            type_=type_,
            detail=detail,
            details=details,
            instance=instance,
        )

        compute_service_error.additional_properties = d
        return compute_service_error

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
