from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1EmptyDirVolumeSource")


@_attrs_define
class Corev1EmptyDirVolumeSource:
    """
    Attributes:
        medium (str | Unset):
        size_limit (None | str | Unset): Kubernetes resource.Quantity (e.g. "500m", "2Gi", "4").
    """

    medium: str | Unset = UNSET
    size_limit: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        medium = self.medium

        size_limit: None | str | Unset
        if isinstance(self.size_limit, Unset):
            size_limit = UNSET
        else:
            size_limit = self.size_limit

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if medium is not UNSET:
            field_dict["medium"] = medium
        if size_limit is not UNSET:
            field_dict["sizeLimit"] = size_limit

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        medium = d.pop("medium", UNSET)

        def _parse_size_limit(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        size_limit = _parse_size_limit(d.pop("sizeLimit", UNSET))

        corev_1_empty_dir_volume_source = cls(
            medium=medium,
            size_limit=size_limit,
        )

        corev_1_empty_dir_volume_source.additional_properties = d
        return corev_1_empty_dir_volume_source

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
