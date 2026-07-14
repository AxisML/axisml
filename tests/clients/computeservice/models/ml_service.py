from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_annotations import MLServiceAnnotations
    from ..models.ml_service_labels import MLServiceLabels
    from ..models.ml_service_spec import MLServiceSpec
    from ..models.ml_service_status import MLServiceStatus


T = TypeVar("T", bound="MLService")


@_attrs_define
class MLService:
    """
    Example:
        {'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-28T09:25:00Z', 'description': 'Llama-3 8B online inference on the vLLM backend.', 'displayName':
            'Llama-3 8B inference service', 'generation': 3, 'id': 'c2e1a0b9-8d7c-6b5a-4f3e-2d1c0b9a8f7e', 'kind':
            'service', 'labels': {'team': 'vision'}, 'name': 'llama3-8b', 'namespace': 'team-vision', 'observedGeneration':
            3, 'owner': 'li.wei', 'phase': 'Ready', 'spec': {'backend': {'engine': 'llminference', 'name': 'kserve'},
            'roles': [{'name': 'predictor', 'replicas': 2, 'template': {'args': ['--model', 'meta-llama/Llama-3-8b', '--max-
            model-len', '8192'], 'image': 'registry.axisml.io/serving/vllm:0.6.2', 'ports': [{'containerPort': 8080, 'name':
            'http', 'protocol': 'TCP'}], 'resources': {'limits': {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'},
            'requests': {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer':
            'https://auth.axisml.io', 'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'}, 'type': 'jwt'}, 'enabled':
            True, 'hostname': 'llama3-8b.team-vision.axisml.io', 'path': '/v1', 'portName': 'http', 'targetRole':
            'predictor'}, 'runPolicy': {'progressDeadlineSeconds': 600}, 'scheduling': {'quota': 'axisml-team-vision-
            gpu-a100'}}, 'status': {'endpoint': 'https://llama3-8b.team-vision.axisml.io/v1', 'message': '2/2 replicas
            ready.', 'readyReplicas': 2}, 'updatedAt': '2026-06-28T09:45:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the service was created.
        generation (int): Desired-state generation, bumped on every spec-affecting change (scale).
        id (UUID): Stable service identifier (PG row UUID).
        kind (str): Service kind (service, workspace, tensorboard).
        name (str): MLService name, unique within the namespace.
        namespace (str): Namespace (= tenant identifier) the service belongs to.
        observed_generation (int): Generation the operator last reconciled; equals generation when in sync.
        phase (str): Current service lifecycle phase: Creating, Pending, Ready, Degraded, Failed, Deleting, Deleted.
        spec (MLServiceSpec):
        status (MLServiceStatus):
        updated_at (datetime.datetime): Time the service was last updated.
        annotations (MLServiceAnnotations | Unset): User-defined annotations.
        deleted_at (datetime.datetime | None | Unset): Soft-deletion timestamp, set once the service is deleted.
        description (str | Unset): Free-text service description.
        display_name (str | Unset): Human-readable service label.
        labels (MLServiceLabels | Unset): User-defined labels.
        owner (str | Unset): Username of the service owner.
    """

    created_at: datetime.datetime
    generation: int
    id: UUID
    kind: str
    name: str
    namespace: str
    observed_generation: int
    phase: str
    spec: MLServiceSpec
    status: MLServiceStatus
    updated_at: datetime.datetime
    annotations: MLServiceAnnotations | Unset = UNSET
    deleted_at: datetime.datetime | None | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: MLServiceLabels | Unset = UNSET
    owner: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        generation = self.generation

        id = str(self.id)

        kind = self.kind

        name = self.name

        namespace = self.namespace

        observed_generation = self.observed_generation

        phase = self.phase

        spec = self.spec.to_dict()

        status = self.status.to_dict()

        updated_at = self.updated_at.isoformat()

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

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        owner = self.owner

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "generation": generation,
                "id": id,
                "kind": kind,
                "name": name,
                "namespace": namespace,
                "observedGeneration": observed_generation,
                "phase": phase,
                "spec": spec,
                "status": status,
                "updatedAt": updated_at,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if deleted_at is not UNSET:
            field_dict["deletedAt"] = deleted_at
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if owner is not UNSET:
            field_dict["owner"] = owner

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_annotations import MLServiceAnnotations
        from ..models.ml_service_labels import MLServiceLabels
        from ..models.ml_service_spec import MLServiceSpec
        from ..models.ml_service_status import MLServiceStatus

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        generation = d.pop("generation")

        id = UUID(d.pop("id"))

        kind = d.pop("kind")

        name = d.pop("name")

        namespace = d.pop("namespace")

        observed_generation = d.pop("observedGeneration")

        phase = d.pop("phase")

        spec = MLServiceSpec.from_dict(d.pop("spec"))

        status = MLServiceStatus.from_dict(d.pop("status"))

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        _annotations = d.pop("annotations", UNSET)
        annotations: MLServiceAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = MLServiceAnnotations.from_dict(_annotations)

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

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: MLServiceLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = MLServiceLabels.from_dict(_labels)

        owner = d.pop("owner", UNSET)

        ml_service = cls(
            created_at=created_at,
            generation=generation,
            id=id,
            kind=kind,
            name=name,
            namespace=namespace,
            observed_generation=observed_generation,
            phase=phase,
            spec=spec,
            status=status,
            updated_at=updated_at,
            annotations=annotations,
            deleted_at=deleted_at,
            description=description,
            display_name=display_name,
            labels=labels,
            owner=owner,
        )

        ml_service.additional_properties = d
        return ml_service

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
