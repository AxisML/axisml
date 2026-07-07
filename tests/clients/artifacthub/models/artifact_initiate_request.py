from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_initiate_request_annotations import (
        ArtifactInitiateRequestAnnotations,
    )
    from ..models.artifact_initiate_request_labels import ArtifactInitiateRequestLabels
    from ..models.artifact_initiate_request_spec import ArtifactInitiateRequestSpec


T = TypeVar("T", bound="ArtifactInitiateRequest")


@_attrs_define
class ArtifactInitiateRequest:
    """
    Example:
        {'annotations': {'git-commit': '8c1f4e2'}, 'description': 'ResNet-50 image-classification model pretrained on
            ImageNet.', 'displayName': 'ResNet-50', 'labels': {'stage': 'production', 'team': 'vision'}, 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters': '25.6M', 'task': 'image-
            classification'}, 'version': '1.4.0', 'visibility': 'tenant'}

    Attributes:
        kind (str): Artifact kind (model, dataset, image); selects the storage backend and spec schema, immutable once
            created.
        spec (ArtifactInitiateRequestSpec): Kind-specific free-form specification of the artifact.
        version (str): Version string to create for the artifact name.
        annotations (ArtifactInitiateRequestAnnotations | Unset): Non-identifying metadata annotations.
        description (str | Unset): Free-form description of the artifact.
        display_name (str | Unset): Human-readable name for display.
        labels (ArtifactInitiateRequestLabels | Unset): K8s-style labels used for selector filtering.
        source (str | Unset): Provenance of the version (webUpload, oras, dockerPush, external); external registers a
            remote artifact with no upload.
        source_uri (str | Unset): Remote URI of the artifact; required when source is external.
        visibility (str | Unset): Access scope of the artifact (tenant or public); defaults to tenant.
    """

    kind: str
    spec: ArtifactInitiateRequestSpec
    version: str
    annotations: ArtifactInitiateRequestAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: ArtifactInitiateRequestLabels | Unset = UNSET
    source: str | Unset = UNSET
    source_uri: str | Unset = UNSET
    visibility: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        spec = self.spec.to_dict()

        version = self.version

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        source = self.source

        source_uri = self.source_uri

        visibility = self.visibility

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
                "spec": spec,
                "version": version,
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
        if source is not UNSET:
            field_dict["source"] = source
        if source_uri is not UNSET:
            field_dict["sourceUri"] = source_uri
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_initiate_request_annotations import (
            ArtifactInitiateRequestAnnotations,
        )
        from ..models.artifact_initiate_request_labels import (
            ArtifactInitiateRequestLabels,
        )
        from ..models.artifact_initiate_request_spec import ArtifactInitiateRequestSpec

        d = dict(src_dict)
        kind = d.pop("kind")

        spec = ArtifactInitiateRequestSpec.from_dict(d.pop("spec"))

        version = d.pop("version")

        _annotations = d.pop("annotations", UNSET)
        annotations: ArtifactInitiateRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ArtifactInitiateRequestAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: ArtifactInitiateRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = ArtifactInitiateRequestLabels.from_dict(_labels)

        source = d.pop("source", UNSET)

        source_uri = d.pop("sourceUri", UNSET)

        visibility = d.pop("visibility", UNSET)

        artifact_initiate_request = cls(
            kind=kind,
            spec=spec,
            version=version,
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            source=source,
            source_uri=source_uri,
            visibility=visibility,
        )

        artifact_initiate_request.additional_properties = d
        return artifact_initiate_request

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
