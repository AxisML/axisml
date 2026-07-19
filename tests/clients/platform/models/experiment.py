from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job_spec import JobSpec
    from ..models.run_summary import RunSummary
    from ..models.string_map import StringMap


T = TypeVar("T", bound="Experiment")


@_attrs_define
class Experiment:
    r"""
    Example:
        {'annotations': {'axisml.io/created-by': 'li.wei'}, 'createdAt': '2026-06-20T08:00:00Z', 'description':
            'Training experiment fine-tuning BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning experiment', 'id':
            'd4f8a1b2-3c5e-4a7b-9c0d-1e2f3a4b5c6d', 'labels': {'team': 'nlp'}, 'name': 'bert-finetune', 'namespace': 'team-
            nlp', 'owner': 'zhang.san', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'runSummary': {'active': 1,
            'count': 5, 'latestPhase': 'Succeeded', 'latestRunAt': '2026-06-28T09:25:00Z', 'recent': ['Succeeded',
            'Running', 'Succeeded', 'Failed', 'Succeeded']}, 'spec': {'artifacts': [{'kind': 'model', 'name': 'bert-base',
            'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-trainer'}, 'poolName': 'gpu-a100',
            'roles': [{'name': 'worker', 'replicas': 2, 'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001',
            '--epochs', '10', '--batch-size', '64'], 'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE',
            'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources': {'cpu': '8', 'memory':
            '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}, 'tenantName': 'team-nlp', 'updatedAt':
            '2026-06-28T09:30:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the experiment was created.
        id (UUID): Stable experiment identifier.
        name (str): Experiment definition name (unique within the tenant).
        namespace (str): Platform tenant namespace the experiment belongs to.
        owner (str): Username of the experiment owner.
        spec (JobSpec):  Example: {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version': '1.4.0'}], 'backend':
            {'engine': 'pytorchjob', 'name': 'native'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'resnet-training-config'}], 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'envFrom': [{'configMapRef': {'name':
            'resnet-training-config'}}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort':
            8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'},
            'volumeMounts': [{'mountPath': '/data', 'name': 'data'}, {'mountPath': '/etc/axisml', 'name': 'config',
            'readOnly': True}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}},
            {'configMap': {'name': 'resnet-training-config'}, 'name': 'config'}]}}], 'runPolicy': {'activeDeadlineSeconds':
            86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'unitName':
            'a100-2x'}.
        tenant_name (str): Tenant identifier owning the experiment.
        updated_at (datetime.datetime): Time the experiment was last updated.
        annotations (StringMap | Unset):
        description (str | Unset): Free-text experiment description.
        display_name (str | Unset): Human-readable experiment label.
        labels (StringMap | Unset):
        owner_id (UUID | Unset): User ID of the experiment owner.
        run_summary (None | RunSummary | Unset): Roll-up of the experiment's Runs (count + recent phases on lists,
            latest phase on detail).
    """

    created_at: datetime.datetime
    id: UUID
    name: str
    namespace: str
    owner: str
    spec: JobSpec
    tenant_name: str
    updated_at: datetime.datetime
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    run_summary: None | RunSummary | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.run_summary import RunSummary

        created_at = self.created_at.isoformat()

        id = str(self.id)

        name = self.name

        namespace = self.namespace

        owner = self.owner

        spec = self.spec.to_dict()

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

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        run_summary: dict[str, Any] | None | Unset
        if isinstance(self.run_summary, Unset):
            run_summary = UNSET
        elif isinstance(self.run_summary, RunSummary):
            run_summary = self.run_summary.to_dict()
        else:
            run_summary = self.run_summary

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "name": name,
                "namespace": namespace,
                "owner": owner,
                "spec": spec,
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
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if run_summary is not UNSET:
            field_dict["runSummary"] = run_summary

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_spec import JobSpec
        from ..models.run_summary import RunSummary
        from ..models.string_map import StringMap

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        owner = d.pop("owner")

        spec = JobSpec.from_dict(d.pop("spec"))

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

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        def _parse_run_summary(data: object) -> None | RunSummary | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                run_summary_type_1 = RunSummary.from_dict(data)

                return run_summary_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | RunSummary | Unset, data)

        run_summary = _parse_run_summary(d.pop("runSummary", UNSET))

        experiment = cls(
            created_at=created_at,
            id=id,
            name=name,
            namespace=namespace,
            owner=owner,
            spec=spec,
            tenant_name=tenant_name,
            updated_at=updated_at,
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            owner_id=owner_id,
            run_summary=run_summary,
        )

        experiment.additional_properties = d
        return experiment

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
