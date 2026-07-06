from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_annotations import ArtifactAnnotations
    from ..models.artifact_labels import ArtifactLabels
    from ..models.artifact_spec import ArtifactSpec


T = TypeVar("T", bound="Artifact")


@_attrs_define
class Artifact:
    """
    Example:
        {'annotations': {'git-commit': '8c1f4e2'}, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'ResNet-50 image-
            classification model pretrained on ImageNet.', 'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10', 'displayName': 'ResNet-50', 'id':
            '8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10', 'kind': 'model', 'labels': {'stage': 'production', 'team': 'vision'},
            'name': 'resnet50', 'namespace': 'team-vision', 'owner': 'li.wei', 'readyAt': '2026-06-28T09:30:00Z', 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters': '25.6M', 'task': 'image-
            classification'}, 'status': 'Ready', 'updatedAt': '2026-06-28T09:30:00Z', 'version': '1.4.0', 'visibility':
            'tenant'}

    Attributes:
        created_at (datetime.datetime): Creation timestamp (RFC3339).
        id (UUID): Stable artifact identifier (UUID).
        kind (str): Artifact kind (model, dataset, image).
        name (str): Artifact name, unique within the namespace across all kinds.
        namespace (str): Tenant namespace the artifact belongs to (= compute tenants.name).
        spec (ArtifactSpec): Kind-specific free-form specification of the artifact.
        status (str): Lifecycle status (Uploading, Ready, Failed, Deleting, Deleted).
        updated_at (datetime.datetime): Last-update timestamp (RFC3339).
        version (str): Artifact version (free-form string).
        visibility (str): Access scope of the artifact (tenant or public).
        annotations (ArtifactAnnotations | Unset): Non-identifying metadata annotations.
        deleted_at (datetime.datetime | None | Unset): Soft-delete timestamp, set when the artifact is being removed
            (RFC3339).
        description (str | Unset): Free-form description of the artifact.
        digest (str | Unset): Content digest of the stored artifact, set once the upload completes.
        display_name (str | Unset): Human-readable name for display.
        labels (ArtifactLabels | Unset): K8s-style labels used for selector filtering.
        message (str | Unset): Human-readable detail for the current status (e.g. failure reason).
        owner (str | Unset): Identity that owns the artifact version.
        ready_at (datetime.datetime | None | Unset): Timestamp the artifact became Ready (RFC3339).
        source (str | Unset): Provenance of the version (webUpload, oras, dockerPush, external).
    """

    created_at: datetime.datetime
    id: UUID
    kind: str
    name: str
    namespace: str
    spec: ArtifactSpec
    status: str
    updated_at: datetime.datetime
    version: str
    visibility: str
    annotations: ArtifactAnnotations | Unset = UNSET
    deleted_at: datetime.datetime | None | Unset = UNSET
    description: str | Unset = UNSET
    digest: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: ArtifactLabels | Unset = UNSET
    message: str | Unset = UNSET
    owner: str | Unset = UNSET
    ready_at: datetime.datetime | None | Unset = UNSET
    source: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = str(self.id)

        kind = self.kind

        name = self.name

        namespace = self.namespace

        spec = self.spec.to_dict()

        status = self.status

        updated_at = self.updated_at.isoformat()

        version = self.version

        visibility = self.visibility

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        deleted_at: None | str | Unset
        if isinstance(self.deleted_at, Unset):
            deleted_at = UNSET
        elif isinstance(self.deleted_at, datetime.datetime):
            deleted_at = self.deleted_at.isoformat()
        else:
            deleted_at = self.deleted_at

        description = self.description

        digest = self.digest

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        message = self.message

        owner = self.owner

        ready_at: None | str | Unset
        if isinstance(self.ready_at, Unset):
            ready_at = UNSET
        elif isinstance(self.ready_at, datetime.datetime):
            ready_at = self.ready_at.isoformat()
        else:
            ready_at = self.ready_at

        source = self.source

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "name": name,
                "namespace": namespace,
                "spec": spec,
                "status": status,
                "updatedAt": updated_at,
                "version": version,
                "visibility": visibility,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if deleted_at is not UNSET:
            field_dict["deletedAt"] = deleted_at
        if description is not UNSET:
            field_dict["description"] = description
        if digest is not UNSET:
            field_dict["digest"] = digest
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if message is not UNSET:
            field_dict["message"] = message
        if owner is not UNSET:
            field_dict["owner"] = owner
        if ready_at is not UNSET:
            field_dict["readyAt"] = ready_at
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_annotations import ArtifactAnnotations
        from ..models.artifact_labels import ArtifactLabels
        from ..models.artifact_spec import ArtifactSpec

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        kind = d.pop("kind")

        name = d.pop("name")

        namespace = d.pop("namespace")

        spec = ArtifactSpec.from_dict(d.pop("spec"))

        status = d.pop("status")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        version = d.pop("version")

        visibility = d.pop("visibility")

        _annotations = d.pop("annotations", UNSET)
        annotations: ArtifactAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ArtifactAnnotations.from_dict(_annotations)

        def _parse_deleted_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                deleted_at_type_0 = datetime.datetime.fromisoformat(data)

                return deleted_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        deleted_at = _parse_deleted_at(d.pop("deletedAt", UNSET))

        description = d.pop("description", UNSET)

        digest = d.pop("digest", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: ArtifactLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = ArtifactLabels.from_dict(_labels)

        message = d.pop("message", UNSET)

        owner = d.pop("owner", UNSET)

        def _parse_ready_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                ready_at_type_0 = datetime.datetime.fromisoformat(data)

                return ready_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        ready_at = _parse_ready_at(d.pop("readyAt", UNSET))

        source = d.pop("source", UNSET)

        artifact = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            name=name,
            namespace=namespace,
            spec=spec,
            status=status,
            updated_at=updated_at,
            version=version,
            visibility=visibility,
            annotations=annotations,
            deleted_at=deleted_at,
            description=description,
            digest=digest,
            display_name=display_name,
            labels=labels,
            message=message,
            owner=owner,
            ready_at=ready_at,
            source=source,
        )

        artifact.additional_properties = d
        return artifact

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
