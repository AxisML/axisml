from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_definition_kind import ArtifactDefinitionKind
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_definition_spec import ArtifactDefinitionSpec
    from ..models.string_map import StringMap


T = TypeVar("T", bound="ArtifactDefinition")


@_attrs_define
class ArtifactDefinition:
    """
    Example:
        {'annotations': {'git-commit': '8c1f4e2'}, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'ResNet-50 image-
            classification model.', 'displayName': 'ResNet-50', 'id': '1f2e3d4c-5b6a-7980-abcd-ef0123456789', 'kind':
            'model', 'labels': {'team': 'vision'}, 'latestVersion': '1.4.0', 'latestVersionAt': '2026-06-28T09:30:00Z',
            'name': 'resnet50', 'namespace': 'team-vision', 'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-
            ef0123456789', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters': '25.6M', 'task': 'image-
            classification'}, 'tenantName': 'team-vision', 'updatedAt': '2026-06-28T09:30:00Z', 'versionCount': 3}

    Attributes:
        created_at (datetime.datetime): Time the definition was created.
        id (UUID): Stable definition identifier.
        kind (ArtifactDefinitionKind): Definition kind (model or image).
        name (str): Definition name (unique within the tenant).
        namespace (str): Platform tenant namespace the definition belongs to.
        tenant_name (str): Tenant identifier owning the definition.
        updated_at (datetime.datetime): Time the definition was last updated.
        annotations (StringMap | Unset):
        description (str | Unset): Free-text definition description.
        display_name (str | Unset): Human-readable definition label.
        labels (StringMap | Unset):
        latest_version (str | Unset): Version string of the most recent version.
        latest_version_at (datetime.datetime | None | Unset): Creation time of the most recent version.
        owner (str | Unset): Username of the definition owner.
        owner_id (UUID | Unset): User ID of the definition owner.
        spec (ArtifactDefinitionSpec | Unset): Pass-through definition spec (free-form).
        version_count (int | Unset): Number of versions under the definition (roll-up for list cards).
    """

    created_at: datetime.datetime
    id: UUID
    kind: ArtifactDefinitionKind
    name: str
    namespace: str
    tenant_name: str
    updated_at: datetime.datetime
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    latest_version: str | Unset = UNSET
    latest_version_at: datetime.datetime | None | Unset = UNSET
    owner: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    spec: ArtifactDefinitionSpec | Unset = UNSET
    version_count: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = str(self.id)

        kind = self.kind.value

        name = self.name

        namespace = self.namespace

        tenant_name = self.tenant_name

        updated_at = self.updated_at.isoformat()

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        latest_version = self.latest_version

        latest_version_at: None | str | Unset
        if isinstance(self.latest_version_at, Unset):
            latest_version_at = UNSET
        elif isinstance(self.latest_version_at, datetime.datetime):
            latest_version_at = self.latest_version_at.isoformat()
        else:
            latest_version_at = self.latest_version_at

        owner = self.owner

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        version_count = self.version_count

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "name": name,
                "namespace": namespace,
                "tenantName": tenant_name,
                "updatedAt": updated_at,
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
        if latest_version is not UNSET:
            field_dict["latestVersion"] = latest_version
        if latest_version_at is not UNSET:
            field_dict["latestVersionAt"] = latest_version_at
        if owner is not UNSET:
            field_dict["owner"] = owner
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if spec is not UNSET:
            field_dict["spec"] = spec
        if version_count is not UNSET:
            field_dict["versionCount"] = version_count

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_definition_spec import ArtifactDefinitionSpec
        from ..models.string_map import StringMap

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        kind = ArtifactDefinitionKind(d.pop("kind"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        tenant_name = d.pop("tenantName")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

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

        latest_version = d.pop("latestVersion", UNSET)

        def _parse_latest_version_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                latest_version_at_type_0 = datetime.datetime.fromisoformat(data)

                return latest_version_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        latest_version_at = _parse_latest_version_at(d.pop("latestVersionAt", UNSET))

        owner = d.pop("owner", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        _spec = d.pop("spec", UNSET)
        spec: ArtifactDefinitionSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = ArtifactDefinitionSpec.from_dict(_spec)

        version_count = d.pop("versionCount", UNSET)

        artifact_definition = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            name=name,
            namespace=namespace,
            tenant_name=tenant_name,
            updated_at=updated_at,
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            latest_version=latest_version,
            latest_version_at=latest_version_at,
            owner=owner,
            owner_id=owner_id,
            spec=spec,
            version_count=version_count,
        )

        artifact_definition.additional_properties = d
        return artifact_definition

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
