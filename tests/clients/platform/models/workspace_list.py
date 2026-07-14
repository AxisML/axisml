from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workspace import Workspace


T = TypeVar("T", bound="WorkspaceList")


@_attrs_define
class WorkspaceList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'args': ['--NotebookApp.token='], 'command': ['start-notebook.sh'],
            'computeNamespace': 'axisml-team-vision', 'containerPort': 8888, 'createdAt': '2026-06-20T08:00:00Z',
            'description': 'JupyterLab interactive development environment.', 'desiredState': 'Running', 'displayName':
            'Vision team dev environment', 'endpoint': {'accessUrl': 'https://axisml.example.com/ws/team-vision/notebook-
            dev/', 'internalDns': 'notebook-dev.axisml-team-vision.svc.cluster.local', 'tools': [{'label': 'JupyterLab',
            'name': 'jupyter', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/lab'}, {'label': 'VS Code',
            'name': 'vscode', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/vscode/'}, {'label':
            'Terminal', 'name': 'terminal', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/terminals/1'}]},
            'env': [{'name': 'JUPYTER_ENABLE_LAB', 'value': 'yes'}], 'id': 'f1e2d3c4-5b6a-4798-8c0d-1e2f3a4b5c6d', 'image':
            'registry.axisml.io/dev/jupyter:3.0.0', 'lastStartedAt': '2026-06-28T09:00:00Z', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'message': 'Workspace is ready.', 'name': 'notebook-dev', 'namespace': 'team-
            vision', 'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'phase': 'Running', 'poolName':
            'gpu-a100', 'readyReplicas': 1, 'replicas': 1, 'resources': {'cpu': '4', 'memory': '32Gi', 'nvidia.com/gpu':
            '1'}, 'tenantDisplayName': 'Vision Team', 'tenantName': 'team-vision', 'unitName': 'a100-1x', 'updatedAt':
            '2026-06-28T09:30:00Z', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data', 'used':
            '12Gi'}]}], 'partial': False}

    Attributes:
        count (int): Number of workspaces in this page.
        items (list[Workspace]): Workspaces in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[Workspace]
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
        from ..models.workspace import Workspace

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Workspace.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        workspace_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        workspace_list.additional_properties = d
        return workspace_list

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
