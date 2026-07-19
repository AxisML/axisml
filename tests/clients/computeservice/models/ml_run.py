from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_annotations import MLRunAnnotations
    from ..models.ml_run_labels import MLRunLabels
    from ..models.ml_run_spec import MLRunSpec
    from ..models.ml_run_status import MLRunStatus


T = TypeVar("T", bound="MLRun")


@_attrs_define
class MLRun:
    r"""
    Example:
        {'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-28T09:25:00Z', 'description': 'Distributed ResNet-50 training on ImageNet.', 'displayName': 'ResNet-50
            Training #7', 'id': 'b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5', 'labels': {'team': 'vision'}, 'name': 'resnet-
            train-7', 'namespace': 'team-vision', 'owner': 'li.wei', 'phase': 'Running', 'spec': {'backend': {'engine':
            'pytorchjob', 'name': 'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'trainer-config'}], 'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure',
            'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'], 'env':
            [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources':
            {'limits': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory': '64Gi',
            'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'ttlSecondsAfterFinished': 3600}, 'scheduling': {'priorityClass': 'high-priority', 'quota': 'axisml-team-vision-
            gpu-a100'}}, 'status': {'message': 'All worker replicas ready.', 'startedAt': '2026-06-28T09:30:00Z'},
            'updatedAt': '2026-06-28T09:45:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the run was created.
        id (UUID): Stable run identifier (PG row UUID).
        name (str): MLRun name, unique within the namespace.
        namespace (str): Namespace (= tenant identifier) the run belongs to.
        phase (str): Current run lifecycle phase: Creating, Pending, Running, Succeeded, Failed, Canceling, Cancelled,
            Deleting, Deleted.
        spec (MLRunSpec):
        status (MLRunStatus):
        updated_at (datetime.datetime): Time the run was last updated.
        annotations (MLRunAnnotations | Unset): User-defined annotations.
        deleted_at (datetime.datetime | None | Unset): Soft-deletion timestamp, set once the run is deleted.
        description (str | Unset): Free-text run description.
        display_name (str | Unset): Human-readable run label.
        labels (MLRunLabels | Unset): User-defined labels.
        owner (str | Unset): Username of the run owner.
    """

    created_at: datetime.datetime
    id: UUID
    name: str
    namespace: str
    phase: str
    spec: MLRunSpec
    status: MLRunStatus
    updated_at: datetime.datetime
    annotations: MLRunAnnotations | Unset = UNSET
    deleted_at: datetime.datetime | None | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: MLRunLabels | Unset = UNSET
    owner: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = str(self.id)

        name = self.name

        namespace = self.namespace

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
                "id": id,
                "name": name,
                "namespace": namespace,
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
        from ..models.ml_run_annotations import MLRunAnnotations
        from ..models.ml_run_labels import MLRunLabels
        from ..models.ml_run_spec import MLRunSpec
        from ..models.ml_run_status import MLRunStatus

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        phase = d.pop("phase")

        spec = MLRunSpec.from_dict(d.pop("spec"))

        status = MLRunStatus.from_dict(d.pop("status"))

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        _annotations = d.pop("annotations", UNSET)
        annotations: MLRunAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = MLRunAnnotations.from_dict(_annotations)

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
        labels: MLRunLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = MLRunLabels.from_dict(_labels)

        owner = d.pop("owner", UNSET)

        ml_run = cls(
            created_at=created_at,
            id=id,
            name=name,
            namespace=namespace,
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

        ml_run.additional_properties = d
        return ml_run

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
