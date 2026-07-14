from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job import Job


T = TypeVar("T", bound="JobList")


@_attrs_define
class JobList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit':
            '8c1f4e2'}, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'Distributed ResNet-50 training job on
            ImageNet.', 'displayName': 'ResNet-50 Training', 'id': '8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10', 'labels':
            {'team': 'vision'}, 'name': 'resnet-train', 'namespace': 'team-vision', 'owner': 'li.wei', 'ownerId':
            '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'runSummary': {'active': 1, 'count': 7, 'latestPhase': 'Running',
            'latestRunAt': '2026-06-28T09:00:00Z', 'recent': ['Succeeded', 'Failed', 'Succeeded', 'Succeeded', 'Running']},
            'spec': {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version': '1.4.0'}], 'backend': {'engine':
            'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds':
            600, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}, 'tenantName': 'team-vision', 'updatedAt':
            '2026-06-28T09:30:00Z'}], 'partial': False}

    Attributes:
        count (int): Number of jobs in this page.
        items (list[Job]): Jobs in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[Job]
    continue_token: str | Unset = UNSET
    partial: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        continue_token = self.continue_token

        partial = self.partial

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
            }
        )
        if continue_token is not UNSET:
            field_dict["continueToken"] = continue_token
        if partial is not UNSET:
            field_dict["partial"] = partial

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job import Job

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Job.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        job_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        job_list.additional_properties = d
        return job_list

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
