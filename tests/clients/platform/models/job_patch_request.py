from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job_spec import JobSpec
    from ..models.string_map import StringMap


T = TypeVar("T", bound="JobPatchRequest")


@_attrs_define
class JobPatchRequest:
    """
    Example:
        {'description': 'Updated description.', 'displayName': 'ResNet-50 Training (v2)'}

    Attributes:
        annotations (StringMap | Unset):
        description (str | Unset): Updated free-text job description.
        display_name (str | Unset): Updated human-readable job label.
        labels (StringMap | Unset):
        spec (JobSpec | Unset):  Example: {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version': '1.4.0'}],
            'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100', 'quota': 'team-vision', 'roles':
            [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--
            batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}],
            'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http',
            'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts':
            [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName':
            'resnet-imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.
    """

    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    spec: JobSpec | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if spec is not UNSET:
            field_dict["spec"] = spec

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_spec import JobSpec
        from ..models.string_map import StringMap

        d = dict(src_dict)
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

        _spec = d.pop("spec", UNSET)
        spec: JobSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = JobSpec.from_dict(_spec)

        job_patch_request = cls(
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            spec=spec,
        )

        job_patch_request.additional_properties = d
        return job_patch_request

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
