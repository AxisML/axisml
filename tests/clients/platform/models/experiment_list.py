from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.experiment import Experiment


T = TypeVar("T", bound="ExperimentList")


@_attrs_define
class ExperimentList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'annotations': {'axisml.io/created-by': 'li.wei'}, 'createdAt':
            '2026-06-20T08:00:00Z', 'description': 'Training experiment fine-tuning BERT on a Chinese corpus.',
            'displayName': 'BERT fine-tuning experiment', 'id': 'd4f8a1b2-3c5e-4a7b-9c0d-1e2f3a4b5c6d', 'labels': {'team':
            'nlp'}, 'name': 'bert-finetune', 'namespace': 'team-nlp', 'owner': 'zhang.san', 'ownerId':
            '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'runSummary': {'active': 1, 'count': 5, 'latestPhase': 'Succeeded',
            'latestRunAt': '2026-06-28T09:25:00Z', 'recent': ['Succeeded', 'Running', 'Succeeded', 'Failed', 'Succeeded']},
            'spec': {'artifacts': [{'kind': 'model', 'name': 'bert-base', 'version': '2.1.0'}], 'backend': {'engine':
            'pytorchjob', 'name': 'kubeflow-trainer'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001', '--epochs', '10', '--batch-size', '64'],
            'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE', 'value': 'offline'}], 'image':
            'registry.axisml.io/training/bert:2.1.0', 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}],
            'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName':
            'a100-2x'}, 'tenantName': 'team-nlp', 'updatedAt': '2026-06-28T09:30:00Z'}], 'partial': False}

    Attributes:
        count (int): Number of experiments in this page.
        items (list[Experiment]): Experiments in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[Experiment]
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
        from ..models.experiment import Experiment

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Experiment.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        experiment_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        experiment_list.additional_properties = d
        return experiment_list

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
