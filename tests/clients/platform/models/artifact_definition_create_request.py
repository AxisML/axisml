from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_definition_create_request_spec import (
        ArtifactDefinitionCreateRequestSpec,
    )
    from ..models.string_map import StringMap


T = TypeVar("T", bound="ArtifactDefinitionCreateRequest")


@_attrs_define
class ArtifactDefinitionCreateRequest:
    """
    Example:
        {'description': 'ResNet-50 image-classification model.', 'displayName': 'ResNet-50', 'labels': {'team':
            'vision'}, 'name': 'resnet50'}

    Attributes:
        name (str): Artifact definition name (unique within the tenant).
        annotations (StringMap | Unset):
        description (str | Unset): Free-text definition description.
        display_name (str | Unset): Human-readable definition label.
        labels (StringMap | Unset):
        spec (ArtifactDefinitionCreateRequestSpec | Unset): Pass-through definition spec (free-form).
    """

    name: str
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    spec: ArtifactDefinitionCreateRequestSpec | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if spec is not UNSET:
            field_dict["spec"] = spec

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_definition_create_request_spec import (
            ArtifactDefinitionCreateRequestSpec,
        )
        from ..models.string_map import StringMap

        d = dict(src_dict)
        name = d.pop("name")

        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        _spec = d.pop("spec", UNSET)
        spec: ArtifactDefinitionCreateRequestSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = ArtifactDefinitionCreateRequestSpec.from_dict(_spec)

        artifact_definition_create_request = cls(
            name=name,
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            spec=spec,
        )

        artifact_definition_create_request.additional_properties = d
        return artifact_definition_create_request

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
