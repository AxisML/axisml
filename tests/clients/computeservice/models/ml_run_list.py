from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run import MLRun


T = TypeVar("T", bound="MLRunList")


@_attrs_define
class MLRunList:
    """
    Example:
        {'items': [{'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-28T09:25:00Z', 'description': 'Distributed ResNet-50 training on ImageNet.', 'displayName': 'ResNet-50
            Training #7', 'id': 'b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5', 'labels': {'team': 'vision'}, 'name': 'resnet-
            train-7', 'namespace': 'team-vision', 'owner': 'li.wei', 'phase': 'Running', 'spec': {'backend': {'engine':
            'pytorchjob', 'name': 'kubeflow-trainer'}, 'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy':
            'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0',
            'resources': {'limits': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8',
            'memory': '64Gi', 'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'ttlSecondsAfterFinished': 3600}, 'scheduling': {'priorityClass': 'high-priority', 'quota': 'axisml-team-vision-
            gpu-a100'}}, 'status': {'message': 'All worker replicas ready.', 'startedAt': '2026-06-28T09:30:00Z'},
            'updatedAt': '2026-06-28T09:45:00Z'}], 'total': 1}

    Attributes:
        count (int): Number of items returned in this page (len(items)).
        items (list[MLRun]): The page of items for the current offset.
        total (int): Total number of matching items across all pages.
        continue_token (str | Unset): Kubernetes-style continuation token for the next page; empty/absent on the final
            page.
    """

    count: int
    items: list[MLRun]
    total: int
    continue_token: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        total = self.total

        continue_token = self.continue_token

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
                "total": total,
            }
        )
        if continue_token is not UNSET:
            field_dict["continueToken"] = continue_token

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_run import MLRun

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = MLRun.from_dict(items_item_data)

            items.append(items_item)

        total = d.pop("total")

        continue_token = d.pop("continueToken", UNSET)

        ml_run_list = cls(
            count=count,
            items=items,
            total=total,
            continue_token=continue_token,
        )

        ml_run_list.additional_properties = d
        return ml_run_list

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
