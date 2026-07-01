from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.traffic_policy import TrafficPolicy


T = TypeVar("T", bound="TrafficPolicyList")


@_attrs_define
class TrafficPolicyList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'accessUrl': 'https://infer.axisml.io/services/team-vision/resnet-
            serving/', 'backends': [{'actualPct': 90, 'ready': True, 'role': 'stable', 'serviceName': 'resnet-serving-v1',
            'weight': 90}, {'actualPct': 10, 'ready': True, 'role': 'canary', 'serviceName': 'resnet-serving-v2', 'weight':
            10}], 'canaryPercent': 10, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'Canary traffic split for the
            ResNet-50 online inference service.', 'displayName': 'ResNet inference traffic', 'endpoint': {'hostname':
            'infer.axisml.io', 'path': '/services/team-vision/resnet-serving/'}, 'id':
            'd4e5f6a7-8b9c-0d1e-2f3a-4b5c6d7e8f90', 'message': 'Routing 90/10 between stable and canary.', 'mode': 'canary',
            'name': 'resnet-serving', 'namespace': 'team-vision', 'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-
            ef0123456789', 'phase': 'Ready', 'tenantDisplayName': 'Vision Team', 'tenantName': 'team-vision', 'updatedAt':
            '2026-06-28T09:30:00Z'}], 'partial': False}

    Attributes:
        count (int): Number of policies in this page.
        items (list[TrafficPolicy]): Traffic policies in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[TrafficPolicy]
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
        from ..models.traffic_policy import TrafficPolicy

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = TrafficPolicy.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        traffic_policy_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        traffic_policy_list.additional_properties = d
        return traffic_policy_list

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
