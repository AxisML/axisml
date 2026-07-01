from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.image import Image


T = TypeVar("T", bound="ImageList")


@_attrs_define
class ImageList:
    """
    Example:
        {'continueToken': '', 'count': 1, 'items': [{'createdAt': '2026-06-20T08:00:00Z', 'description': 'Training image
            with PyTorch 2.3 and CUDA 12.', 'digest':
            'sha256:7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b', 'displayName': 'PyTorch Training
            2.3.0', 'id': 'a1b2c3d4-5e6f-7081-92a3-b4c5d6e7f809', 'name': 'pytorch-train', 'namespace': 'team-vision',
            'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'readyAt': '2026-06-28T09:30:00Z',
            'sizeBytes': 5368709120, 'source': 'dockerPush', 'spec': {'baseImage': 'nvidia/cuda:12.1.0-runtime', 'purpose':
            'training', 'python': '3.11'}, 'status': 'Ready', 'tenantName': 'team-vision', 'updatedAt':
            '2026-06-28T09:30:00Z', 'uri': 'oci://registry.axisml.io/team-vision/pytorch-train:2.3.0', 'version': '2.3.0'}],
            'partial': False}

    Attributes:
        count (int): Number of image versions in this page.
        items (list[Image]): Image versions in this page.
        continue_token (str | Unset): Opaque token to fetch the next page.
        partial (bool | Unset): True if the list was truncated by an upstream limit.
    """

    count: int
    items: list[Image]
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
        from ..models.image import Image

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = Image.from_dict(items_item_data)

            items.append(items_item)

        continue_token = d.pop("continueToken", UNSET)

        partial = d.pop("partial", UNSET)

        image_list = cls(
            count=count,
            items=items,
            continue_token=continue_token,
            partial=partial,
        )

        image_list.additional_properties = d
        return image_list

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
