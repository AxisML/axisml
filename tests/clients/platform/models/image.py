from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_source import ArtifactSource
from ..models.image_status import ImageStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.image_spec import ImageSpec


T = TypeVar("T", bound="Image")


@_attrs_define
class Image:
    """
    Example:
        {'createdAt': '2026-06-20T08:00:00Z', 'description': 'Training image with PyTorch 2.3 and CUDA 12.', 'digest':
            'sha256:7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b', 'displayName': 'PyTorch Training
            2.3.0', 'id': 'a1b2c3d4-5e6f-7081-92a3-b4c5d6e7f809', 'name': 'pytorch-train', 'namespace': 'team-vision',
            'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'readyAt': '2026-06-28T09:30:00Z',
            'sizeBytes': 5368709120, 'source': 'dockerPush', 'spec': {'baseImage': 'nvidia/cuda:12.1.0-runtime', 'purpose':
            'training', 'python': '3.11'}, 'status': 'Ready', 'tenantName': 'team-vision', 'updatedAt':
            '2026-06-28T09:30:00Z', 'uri': 'oci://registry.axisml.io/team-vision/pytorch-train:2.3.0', 'version': '2.3.0'}

    Attributes:
        created_at (datetime.datetime): Time the version was created.
        id (UUID): Stable image version identifier.
        name (str): Image definition name (unique within the tenant).
        namespace (str): Platform tenant namespace the image belongs to.
        status (ImageStatus): Mirrors artifacts ArtifactStatus for kind=image.
        tenant_name (str): Tenant identifier owning the image.
        version (str): Image version (tag).
        description (str | Unset): Free-text version description.
        digest (str | Unset): Content digest of the image.
        display_name (str | Unset): Human-readable version label.
        owner (str | Unset): Username of the version owner.
        owner_id (UUID | Unset): User ID of the version owner.
        ready_at (datetime.datetime | None | Unset): Time the version became Ready.
        size_bytes (int | Unset): Total size of the image content in bytes.
        source (ArtifactSource | Unset): How an artifact version was added.
        spec (ImageSpec | Unset): Artifact-side spec for kind=image; pass-through to artifacts.
        updated_at (datetime.datetime | Unset): Time the version was last updated.
        uri (str | Unset): Artifact registry URI of the version.
    """

    created_at: datetime.datetime
    id: UUID
    name: str
    namespace: str
    status: ImageStatus
    tenant_name: str
    version: str
    description: str | Unset = UNSET
    digest: str | Unset = UNSET
    display_name: str | Unset = UNSET
    owner: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    ready_at: datetime.datetime | None | Unset = UNSET
    size_bytes: int | Unset = UNSET
    source: ArtifactSource | Unset = UNSET
    spec: ImageSpec | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    uri: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = str(self.id)

        name = self.name

        namespace = self.namespace

        status = self.status.value

        tenant_name = self.tenant_name

        version = self.version

        description = self.description

        digest = self.digest

        display_name = self.display_name

        owner = self.owner

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        ready_at: None | str | Unset
        if isinstance(self.ready_at, Unset):
            ready_at = UNSET
        elif isinstance(self.ready_at, datetime.datetime):
            ready_at = self.ready_at.isoformat()
        else:
            ready_at = self.ready_at

        size_bytes = self.size_bytes

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        uri = self.uri

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "name": name,
                "namespace": namespace,
                "status": status,
                "tenantName": tenant_name,
                "version": version,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if digest is not UNSET:
            field_dict["digest"] = digest
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if owner is not UNSET:
            field_dict["owner"] = owner
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if ready_at is not UNSET:
            field_dict["readyAt"] = ready_at
        if size_bytes is not UNSET:
            field_dict["sizeBytes"] = size_bytes
        if source is not UNSET:
            field_dict["source"] = source
        if spec is not UNSET:
            field_dict["spec"] = spec
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if uri is not UNSET:
            field_dict["uri"] = uri

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.image_spec import ImageSpec

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        status = ImageStatus(d.pop("status"))

        tenant_name = d.pop("tenantName")

        version = d.pop("version")

        description = d.pop("description", UNSET)

        digest = d.pop("digest", UNSET)

        display_name = d.pop("displayName", UNSET)

        owner = d.pop("owner", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

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

        size_bytes = d.pop("sizeBytes", UNSET)

        _source = d.pop("source", UNSET)
        source: ArtifactSource | Unset
        if isinstance(_source, Unset):
            source = UNSET
        else:
            source = ArtifactSource(_source)

        _spec = d.pop("spec", UNSET)
        spec: ImageSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = ImageSpec.from_dict(_spec)

        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        uri = d.pop("uri", UNSET)

        image = cls(
            created_at=created_at,
            id=id,
            name=name,
            namespace=namespace,
            status=status,
            tenant_name=tenant_name,
            version=version,
            description=description,
            digest=digest,
            display_name=display_name,
            owner=owner,
            owner_id=owner_id,
            ready_at=ready_at,
            size_bytes=size_bytes,
            source=source,
            spec=spec,
            updated_at=updated_at,
            uri=uri,
        )

        image.additional_properties = d
        return image

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
