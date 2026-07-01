from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1FlockerVolumeSource")


@_attrs_define
class Corev1FlockerVolumeSource:
    """
    Attributes:
        dataset_name (str | Unset):
        dataset_uuid (str | Unset):
    """

    dataset_name: str | Unset = UNSET
    dataset_uuid: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        dataset_name = self.dataset_name

        dataset_uuid = self.dataset_uuid

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if dataset_name is not UNSET:
            field_dict["datasetName"] = dataset_name
        if dataset_uuid is not UNSET:
            field_dict["datasetUUID"] = dataset_uuid

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dataset_name = d.pop("datasetName", UNSET)

        dataset_uuid = d.pop("datasetUUID", UNSET)

        corev_1_flocker_volume_source = cls(
            dataset_name=dataset_name,
            dataset_uuid=dataset_uuid,
        )

        corev_1_flocker_volume_source.additional_properties = d
        return corev_1_flocker_volume_source

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
