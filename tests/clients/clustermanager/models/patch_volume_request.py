from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.patch_volume_request_labels import PatchVolumeRequestLabels


T = TypeVar("T", bound="PatchVolumeRequest")


@_attrs_define
class PatchVolumeRequest:
    """
    Example:
        {'description': 'Shared raw datasets directory (expanded)', 'size': '4Ti'}

    Attributes:
        description (None | str | Unset): Replacement free-text description.
        labels (PatchVolumeRequestLabels | Unset): Replacement user-defined label set.
        size (None | str | Unset): New storage size; expand-only (must be >= current).
    """

    description: None | str | Unset = UNSET
    labels: PatchVolumeRequestLabels | Unset = UNSET
    size: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        description: None | str | Unset
        if isinstance(self.description, Unset):
            description = UNSET
        else:
            description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        size: None | str | Unset
        if isinstance(self.size, Unset):
            size = UNSET
        else:
            size = self.size

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if size is not UNSET:
            field_dict["size"] = size

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.patch_volume_request_labels import PatchVolumeRequestLabels

        d = dict(src_dict)

        def _parse_description(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        description = _parse_description(d.pop("description", UNSET))

        _labels = d.pop("labels", UNSET)
        labels: PatchVolumeRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = PatchVolumeRequestLabels.from_dict(_labels)

        def _parse_size(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        size = _parse_size(d.pop("size", UNSET))

        patch_volume_request = cls(
            description=description,
            labels=labels,
            size=size,
        )

        patch_volume_request.additional_properties = d
        return patch_volume_request

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
