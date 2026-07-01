from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_source import ArtifactSource
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.image_spec import ImageSpec


T = TypeVar("T", bound="ImageInitiateRequest")


@_attrs_define
class ImageInitiateRequest:
    """
    Example:
        {'description': 'Training image with PyTorch 2.3 and CUDA 12.', 'displayName': 'PyTorch Training 2.3.0',
            'source': 'dockerPush', 'spec': {'purpose': 'training'}, 'version': '2.3.0'}

    Attributes:
        spec (ImageSpec): Artifact-side spec for kind=image; pass-through to artifacts.
        version (str): Image version (tag) to create.
        description (str | Unset): Free-text version description.
        display_name (str | Unset): Human-readable version label.
        source (ArtifactSource | Unset): How an artifact version was added.
        source_image_ref (str | Unset): Remote image reference to sync (required when source=external).
    """

    spec: ImageSpec
    version: str
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    source: ArtifactSource | Unset = UNSET
    source_image_ref: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        spec = self.spec.to_dict()

        version = self.version

        description = self.description

        display_name = self.display_name

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value

        source_image_ref = self.source_image_ref

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "spec": spec,
                "version": version,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if source is not UNSET:
            field_dict["source"] = source
        if source_image_ref is not UNSET:
            field_dict["sourceImageRef"] = source_image_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.image_spec import ImageSpec

        d = dict(src_dict)
        spec = ImageSpec.from_dict(d.pop("spec"))

        version = d.pop("version")

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _source = d.pop("source", UNSET)
        source: ArtifactSource | Unset
        if isinstance(_source, Unset):
            source = UNSET
        else:
            source = ArtifactSource(_source)

        source_image_ref = d.pop("sourceImageRef", UNSET)

        image_initiate_request = cls(
            spec=spec,
            version=version,
            description=description,
            display_name=display_name,
            source=source,
            source_image_ref=source_image_ref,
        )

        image_initiate_request.additional_properties = d
        return image_initiate_request

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
