from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1PersistentVolumeClaimVolumeSource")


@_attrs_define
class Corev1PersistentVolumeClaimVolumeSource:
    """
    Attributes:
        claim_name (str):
        read_only (bool | Unset):
    """

    claim_name: str
    read_only: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        claim_name = self.claim_name

        read_only = self.read_only

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "claimName": claim_name,
            }
        )
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        claim_name = d.pop("claimName")

        read_only = d.pop("readOnly", UNSET)

        corev_1_persistent_volume_claim_volume_source = cls(
            claim_name=claim_name,
            read_only=read_only,
        )

        corev_1_persistent_volume_claim_volume_source.additional_properties = d
        return corev_1_persistent_volume_claim_volume_source

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
