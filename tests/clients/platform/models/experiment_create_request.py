from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job_spec import JobSpec
    from ..models.string_map import StringMap


T = TypeVar("T", bound="ExperimentCreateRequest")


@_attrs_define
class ExperimentCreateRequest:
    """
    Example:
        {'description': 'Training experiment fine-tuning BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning
            experiment', 'labels': {'team': 'nlp'}, 'name': 'bert-finetune', 'spec': {'artifacts': [{'kind': 'model',
            'name': 'bert-base', 'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-trainer'},
            'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--lr', '0.001', '--epochs', '10', '--batch-size', '64'], 'command': ['python', 'train.py'], 'env':
            [{'name': 'WANDB_MODE', 'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}}

    Attributes:
        name (str): Experiment definition name (unique within the tenant).
        spec (JobSpec):  Example: {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version': '1.4.0'}], 'backend':
            {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds':
            600, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.
        annotations (StringMap | Unset):
        description (str | Unset): Free-text experiment description.
        display_name (str | Unset): Human-readable experiment label.
        labels (StringMap | Unset):
    """

    name: str
    spec: JobSpec
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        spec = self.spec.to_dict()

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "spec": spec,
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

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_spec import JobSpec
        from ..models.string_map import StringMap

        d = dict(src_dict)
        name = d.pop("name")

        spec = JobSpec.from_dict(d.pop("spec"))

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

        experiment_create_request = cls(
            name=name,
            spec=spec,
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
        )

        experiment_create_request.additional_properties = d
        return experiment_create_request

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
