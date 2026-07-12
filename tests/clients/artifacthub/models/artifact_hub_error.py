from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_hub_error_code import ArtifactHubErrorCode
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_hub_error_details import ArtifactHubErrorDetails


T = TypeVar("T", bound="ArtifactHubError")


@_attrs_define
class ArtifactHubError:
    """
    Example:
        {'code': 'not_found', 'detail': 'artifact team-vision/resnet50@1.4.0 not found', 'instance':
            '/api/v1/namespaces/team-vision/artifacts/resnet50/1.4.0', 'status': 404, 'title': 'artifact not found', 'type':
            'https://axisml.io/errors/not_found'}

    Attributes:
        code (ArtifactHubErrorCode): Discrete business error class.
        status (int): HTTP status code for this occurrence of the problem.
        title (str): Short, human-readable summary of the problem.
        type_ (str): URI reference identifying the problem type.
        detail (str | Unset): Human-readable explanation specific to this occurrence.
        details (ArtifactHubErrorDetails | Unset): Structured, machine-readable detail about the problem.
        instance (str | Unset): URI reference identifying the specific occurrence (the request path).
    """

    code: ArtifactHubErrorCode
    status: int
    title: str
    type_: str
    detail: str | Unset = UNSET
    details: ArtifactHubErrorDetails | Unset = UNSET
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
        from ..models.artifact_hub_error_details import ArtifactHubErrorDetails

        d = dict(src_dict)
        code = ArtifactHubErrorCode(d.pop("code"))

        status = d.pop("status")

        title = d.pop("title")

        type_ = d.pop("type")

        detail = d.pop("detail", UNSET)

        _details = d.pop("details", UNSET)
        details: ArtifactHubErrorDetails | Unset
        if isinstance(_details, Unset):
            details = UNSET
        else:
            details = ArtifactHubErrorDetails.from_dict(_details)

        instance = d.pop("instance", UNSET)

        artifact_hub_error = cls(
            code=code,
            status=status,
            title=title,
            type_=type_,
            detail=detail,
            details=details,
            instance=instance,
        )

        artifact_hub_error.additional_properties = d
        return artifact_hub_error

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
