from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.run import Run


T = TypeVar("T", bound="RunList")


@_attrs_define
class RunList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'backend': {'engine': 'pytorchjob', 'name': 'native'},
            'computeNamespace': 'axisml-team-vision', 'createdAt': '2026-06-28T09:00:00Z', 'description': 'Distributed
            ResNet-50 training run on ImageNet.', 'displayName': 'ResNet-50 Training #7', 'id':
            'b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5', 'jobName': 'resnet-train', 'message': 'All worker replicas ready.',
            'name': 'resnet-train-7', 'namespace': 'team-vision', 'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-
            ef0123456789', 'phase': 'Running', 'poolName': 'gpu-a100', 'resources': {'cpu': '32', 'memory': '256Gi',
            'nvidia.com/gpu': '8'}, 'roles': [{'activeReplicas': 4, 'failedReplicas': 0, 'name': 'worker', 'readyReplicas':
            4, 'replicas': 4, 'restartPolicy': 'OnFailure', 'succeededReplicas': 0, 'template': {'args': ['--epochs', '90',
            '--batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}],
            'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http',
            'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts':
            [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName':
            'resnet-imagenet'}}]}}], 'runNumber': 7, 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'scheduledAt': '2026-06-28T09:00:00Z', 'spec':
            {'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds':
            600, 'ttlSecondsAfterFinished': 3600}, 'scheduling': {'minMember': 4, 'priorityClass': 'high-priority', 'quota':
            'axisml-team-vision-gpu-a100'}}, 'startedAt': '2026-06-28T09:00:00Z', 'tenantDisplayName': 'Vision Team',
            'tenantName': 'team-vision', 'unitName': 'a100-2x', 'updatedAt': '2026-06-28T09:30:00Z'}], 'partial': False}

    Attributes:
        count (int): Number of runs in this page.
        items (list[Run]): Runs in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[Run]
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
        from ..models.run import Run

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Run.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        run_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        run_list.additional_properties = d
        return run_list

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
