from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenant import Tenant


T = TypeVar("T", bound="TenantList")


@_attrs_define
class TenantList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'activeExperimentRuns': 2, 'activeJobRuns': 3, 'annotations':
            {'axisml.io/created-by': 'system-admin'}, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'Computer-vision
            model training and inference team.', 'displayName': 'Vision Team', 'identifier': 'team-vision', 'initResources':
            {'configMaps': [{'name': 'shared-config', 'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-
            system'}}], 'imagePullSecrets': [{'name': 'registry-pull', 'sourceSecretRef': {'name': 'registry-pull',
            'namespace': 'axisml-system'}}], 'secrets': [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-
            key', 'namespace': 'axisml-system'}, 'type': 'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-
            pull'], 'name': 'trainer', 'rbac': {'roleRefs': [{'kind': 'ClusterRole', 'name': 'edit'}]}}]},
            'kubernetesNamespace': 'axisml-team-vision', 'labels': {'team': 'vision'}, 'memberCount': 8, 'onlineServices':
            1, 'owner': 'li.wei', 'phase': 'Active', 'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName':
            'a100-2x'}]}], 'status': {'conditions': [{'lastTransitionTime': '2026-06-28T09:30:00Z', 'message': 'Namespace
            and quota provisioned.', 'reason': 'Provisioned', 'status': 'True', 'type': 'Ready'}], 'message': 'Tenant is
            active.', 'quotas': [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x', 'used': 3}]}]},
            'suspended': False, 'updatedAt': '2026-06-28T09:30:00Z'}], 'partial': False}

    Attributes:
        count (int): Number of tenants in this page.
        items (list[Tenant]): Tenants in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if some rows could not be enriched with live status.
    """

    count: int
    items: list[Tenant]
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
        from ..models.tenant import Tenant

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Tenant.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        tenant_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
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
