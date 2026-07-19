from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service import MLService


T = TypeVar("T", bound="MLServiceList")


@_attrs_define
class MLServiceList:
    r"""
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'accessUrl': 'https://gateway.axisml.io/v1/models/llama3-8b',
            'args': ['--model', 'meta-llama/Llama-3-8b', '--max-model-len', '8192'], 'backend': {'engine': 'llminference',
            'name': 'kserve'}, 'command': ['python', '-m', 'vllm.entrypoints.openai.api_server'], 'computeNamespace':
            'axisml-team-nlp', 'configMaps': [{'data': {'server.yaml': 'maxTokens: 4096\n'}, 'name': 'llama3-serving-
            config'}], 'createdAt': '2026-06-20T08:00:00Z', 'description': 'Llama3-8B online inference service.',
            'desiredState': 'Running', 'displayName': 'Llama3 chat service', 'env': [{'name': 'MAX_TOKENS', 'value':
            '4096'}], 'envFrom': [{'configMapRef': {'name': 'llama3-serving-config'}}], 'id':
            '5d2c9b41-3e8f-4a1c-9d7e-6b4f2a1c8e90', 'image': 'registry.axisml.io/serving/vllm:0.6.0', 'message': 'All
            replicas ready.', 'modelName': 'llama3-8b', 'modelVersion': '1.2.0', 'name': 'llama3-chat', 'namespace': 'team-
            nlp', 'owner': 'zhang.san', 'ownerId': '9f8e7d6c-5b4a-3210-fedc-ba9876543210', 'phase': 'Ready', 'poolName':
            'gpu-a100', 'ports': [{'name': 'http', 'port': 8080}], 'readyReplicas': 3, 'replicas': 3, 'resources': {'cpu':
            '8', 'memory': '64Gi', 'nvidia.com/gpu': '1'}, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'tenantDisplayName': 'Vision Team', 'tenantName': 'team-nlp', 'unitName': 'a100-1x', 'updatedAt':
            '2026-06-28T09:30:00Z', 'volumeMounts': [{'mountPath': '/etc/axisml', 'name': 'config', 'readOnly': True}],
            'volumes': [{'configMap': {'name': 'llama3-serving-config'}, 'name': 'config'}]}], 'partial': False}

    Attributes:
        count (int): Number of services in this page.
        items (list[MLService]): Services in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[MLService]
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
        from ..models.ml_service import MLService

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = MLService.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        ml_service_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        ml_service_list.additional_properties = d
        return ml_service_list

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
