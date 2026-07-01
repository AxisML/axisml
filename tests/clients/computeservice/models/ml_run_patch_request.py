from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_patch_request_annotations import MLRunPatchRequestAnnotations
    from ..models.ml_run_patch_request_labels import MLRunPatchRequestLabels


T = TypeVar("T", bound="MLRunPatchRequest")


@_attrs_define
class MLRunPatchRequest:
    """
    Example:
        {'description': 'Updated description.', 'displayName': 'ResNet-50 Training #7 (rerun)'}

    Attributes:
        annotations (MLRunPatchRequestAnnotations | Unset): Replacement annotation set.
        description (None | str | Unset): Updated free-text run description.
        display_name (None | str | Unset): Updated human-readable run label.
        labels (MLRunPatchRequestLabels | Unset): Replacement label set.
    """

    annotations: MLRunPatchRequestAnnotations | Unset = UNSET
    description: None | str | Unset = UNSET
    display_name: None | str | Unset = UNSET
    labels: MLRunPatchRequestLabels | Unset = UNSET
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
        from ..models.ml_run_patch_request_annotations import (
            MLRunPatchRequestAnnotations,
        )
        from ..models.ml_run_patch_request_labels import MLRunPatchRequestLabels

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: MLRunPatchRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = MLRunPatchRequestAnnotations.from_dict(_annotations)

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
        labels: MLRunPatchRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = MLRunPatchRequestLabels.from_dict(_labels)

        ml_run_patch_request = cls(
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
        )

        ml_run_patch_request.additional_properties = d
        return ml_run_patch_request

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
