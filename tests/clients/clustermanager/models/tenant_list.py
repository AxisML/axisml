from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_tenant import ServerTenant


T = TypeVar("T", bound="TenantList")


@_attrs_define
class TenantList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'annotations': {'resource.axisml.io/last-modified-by': 'li.wei'},
            'createdAt': '2026-06-20T08:00:00Z', 'initResources': {'configMaps': [{'name': 'shared-config',
            'sourceConfigMapRef': {'name': 'tenant-shared-config', 'namespace': 'axisml-system'}}], 'imagePullSecrets':
            [{'name': 'registry-pull', 'sourceSecretRef': {'name': 'registry-pull-credentials', 'namespace': 'axisml-
            system'}}], 'secrets': [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-key', 'namespace':
            'axisml-system'}, 'type': 'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-pull'], 'name':
            'trainer', 'rbac': {'rules': [{'apiGroups': [''], 'resources': ['pods', 'pods/log'], 'verbs': ['get', 'list',
            'watch']}]}}]}, 'labels': {'displayName': 'Vision Team'}, 'name': 'team-vision', 'namespace': {'name': 'team-
            vision'}, 'phase': 'Ready', 'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}],
            'resourceVersion': '184730', 'status': {'message': 'Namespace and quotas provisioned.', 'namespaceReady': True,
            'observedGeneration': 3, 'phase': 'Ready', 'quotas': [{'pool': 'gpu-a100', 'ready': True, 'used': {'cpu': '32',
            'memory': '256Gi', 'nvidia.com/gpu': '4'}}]}}]}

    Attributes:
        count (int): Number of tenants in this page.
        items (list[ServerTenant]): Page of tenants.
        continue_token (str | Unset): Opaque token to fetch the next page; empty when no more pages.
    """

    count: int
    items: list[ServerTenant]
    continue_token: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        continue_token = self.continue_token

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

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_tenant import ServerTenant

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ServerTenant.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        tenant_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
        )

        tenant_list.additional_properties = d
        return tenant_list

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
