from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_source import ArtifactSource
from ..models.remote_source_kind import RemoteSourceKind
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.model_spec import ModelSpec


T = TypeVar("T", bound="ModelInitiateRequest")


@_attrs_define
class ModelInitiateRequest:
    """
    Example:
        {'description': 'ResNet-50 weights fine-tuned on ImageNet.', 'displayName': 'ResNet-50 v1.4.0', 'source':
            'webUpload', 'version': '1.4.0'}

    Attributes:
        version (str): Model version label to create.
        description (str | Unset): Free-text version description.
        display_name (str | Unset): Human-readable version label.
        remote_source_kind (RemoteSourceKind | Unset): Backing store of an externally-registered model version.
        remote_uri (str | Unset): Remote artifact URI to register (required when source=external).
        source (ArtifactSource | Unset): How an artifact version was added.
        spec (ModelSpec | Unset): Artifact-side spec for kind=model; pass-through to artifacts.
    """

    version: str
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    remote_source_kind: RemoteSourceKind | Unset = UNSET
    remote_uri: str | Unset = UNSET
    source: ArtifactSource | Unset = UNSET
    spec: ModelSpec | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        version = self.version

        description = self.description

        display_name = self.display_name

        remote_source_kind: str | Unset = UNSET
        if not isinstance(self.remote_source_kind, Unset):
            remote_source_kind = self.remote_source_kind.value

        remote_uri = self.remote_uri

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "version": version,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if remote_source_kind is not UNSET:
            field_dict["remoteSourceKind"] = remote_source_kind
        if remote_uri is not UNSET:
            field_dict["remoteUri"] = remote_uri
        if source is not UNSET:
            field_dict["source"] = source
        if spec is not UNSET:
            field_dict["spec"] = spec

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.model_spec import ModelSpec

        d = dict(src_dict)
        version = d.pop("version")

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _remote_source_kind = d.pop("remoteSourceKind", UNSET)
        remote_source_kind: RemoteSourceKind | Unset
        if isinstance(_remote_source_kind, Unset):
            remote_source_kind = UNSET
        else:
            remote_source_kind = RemoteSourceKind(_remote_source_kind)

        remote_uri = d.pop("remoteUri", UNSET)

        _source = d.pop("source", UNSET)
        source: ArtifactSource | Unset
        if isinstance(_source, Unset):
            source = UNSET
        else:
            source = ArtifactSource(_source)

        _spec = d.pop("spec", UNSET)
        spec: ModelSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = ModelSpec.from_dict(_spec)

        model_initiate_request = cls(
            version=version,
            description=description,
            display_name=display_name,
            remote_source_kind=remote_source_kind,
            remote_uri=remote_uri,
            source=source,
            spec=spec,
        )

        model_initiate_request.additional_properties = d
        return model_initiate_request

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
