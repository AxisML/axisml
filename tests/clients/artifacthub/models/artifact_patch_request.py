from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_patch_request_annotations import (
        ArtifactPatchRequestAnnotations,
    )
    from ..models.artifact_patch_request_labels import ArtifactPatchRequestLabels


T = TypeVar("T", bound="ArtifactPatchRequest")


@_attrs_define
class ArtifactPatchRequest:
    """
    Example:
        {'annotations': {'reviewed-by': 'zhang.san'}, 'description': 'Updated description.', 'displayName': 'ResNet-50
            (production)', 'labels': {'stage': 'production', 'team': 'vision'}}

    Attributes:
        annotations (ArtifactPatchRequestAnnotations | Unset): Replacement metadata annotations; omit to leave
            unchanged.
        description (None | str | Unset): New free-form description; omit to leave unchanged.
        display_name (None | str | Unset): New human-readable display name; omit to leave unchanged.
        labels (ArtifactPatchRequestLabels | Unset): Replacement K8s-style labels; omit to leave unchanged.
    """

    annotations: ArtifactPatchRequestAnnotations | Unset = UNSET
    description: None | str | Unset = UNSET
    display_name: None | str | Unset = UNSET
    labels: ArtifactPatchRequestLabels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description: None | str | Unset
        if isinstance(self.description, Unset):
            description = UNSET
        else:
            description = self.description

        display_name: None | str | Unset
        if isinstance(self.display_name, Unset):
            display_name = UNSET
        else:
            display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_patch_request_annotations import (
            ArtifactPatchRequestAnnotations,
        )
        from ..models.artifact_patch_request_labels import ArtifactPatchRequestLabels

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: ArtifactPatchRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ArtifactPatchRequestAnnotations.from_dict(_annotations)

        def _parse_description(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        description = _parse_description(d.pop("description", UNSET))

        def _parse_display_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        display_name = _parse_display_name(d.pop("displayName", UNSET))

        _labels = d.pop("labels", UNSET)
        labels: ArtifactPatchRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = ArtifactPatchRequestLabels.from_dict(_labels)

        artifact_patch_request = cls(
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
        )

        artifact_patch_request.additional_properties = d
        return artifact_patch_request

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
