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
        {'items': [{'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-28T09:25:00Z', 'description': 'Canary 10% traffic to v2.', 'displayName': 'Llama-3 canary release',
            'generation': 2, 'id': 'd3f2b1c0-9e8d-7c6b-5a4f-3e2d1c0b9a8f', 'labels': {'team': 'vision'}, 'mode': 'canary',
            'name': 'llama3-canary', 'namespace': 'team-vision', 'observedGeneration': 2, 'owner': 'li.wei', 'phase':
            'Ready', 'spec': {'backend': {'engine': 'inference', 'name': 'kserve'}, 'backends': [{'role': 'stable',
            'serviceName': 'llama3-8b', 'weight': 90}, {'role': 'canary', 'serviceName': 'llama3-8b-v2', 'weight': 10}],
            'endpoint': {'auth': {'jwt': {'audience': 'axisml-inference', 'issuer': 'https://auth.axisml.io', 'jwksUri':
            'https://auth.axisml.io/.well-known/jwks.json'}, 'type': 'jwt'}, 'hostname': 'llama3-8b.team-vision.axisml.io',
            'path': '/v1'}, 'mode': 'canary'}, 'status': {'backends': [{'ready': True, 'serviceName': 'llama3-8b', 'weight':
            90}, {'ready': True, 'serviceName': 'llama3-8b-v2', 'weight': 10}], 'endpoint': 'https://llama3-8b.team-
            vision.axisml.io/v1', 'message': 'Route programmed; weights applied.'}, 'updatedAt': '2026-06-28T09:45:00Z'}],
            'total': 1}

    Attributes:
        count (int): Number of items returned in this page (len(items)).
        items (list[TrafficPolicy]): The page of items for the current offset.
        total (int): Total number of matching items across all pages.
        continue_token (str | Unset): Kubernetes-style continuation token for the next page; empty/absent on the final
            page.
    """

    count: int
    items: list[TrafficPolicy]
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
        from ..models.traffic_policy import TrafficPolicy

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = TrafficPolicy.from_dict(items_item_data)

            items.append(items_item)

        total = d.pop("total")

        continue_token = d.pop("continueToken", UNSET)

        traffic_policy_list = cls(
            count=count,
            items=items,
            total=total,
            continue_token=continue_token,
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
