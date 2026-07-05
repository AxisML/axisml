from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_definition import ArtifactDefinition


T = TypeVar("T", bound="ArtifactDefinitionList")


@_attrs_define
class ArtifactDefinitionList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'annotations': {'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-20T08:00:00Z', 'description': 'ResNet-50 image-classification model.', 'displayName': 'ResNet-50',
            'id': '1f2e3d4c-5b6a-7980-abcd-ef0123456789', 'kind': 'model', 'labels': {'team': 'vision'}, 'latestVersion':
            '1.4.0', 'latestVersionAt': '2026-06-28T09:30:00Z', 'name': 'resnet50', 'namespace': 'team-vision', 'owner':
            'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'spec': {'format': 'safetensors', 'framework':
            'pytorch', 'parameters': '25.6M', 'task': 'image-classification'}, 'tenantName': 'team-vision', 'updatedAt':
            '2026-06-28T09:30:00Z', 'versionCount': 3}], 'partial': False}

    Attributes:
        count (int): Number of definitions in this page.
        items (list[ArtifactDefinition]): Artifact definitions in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[ArtifactDefinition]
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
        from ..models.artifact_definition import ArtifactDefinition

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = ArtifactDefinition.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        artifact_definition_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        artifact_definition_list.additional_properties = d
        return artifact_definition_list

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
