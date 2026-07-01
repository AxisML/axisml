from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.corev_1_volume_source import Corev1VolumeSource


T = TypeVar("T", bound="Corev1Volume")


@_attrs_define
class Corev1Volume:
    """
    Attributes:
        volume_source (Corev1VolumeSource):
        name (str):
    """

    volume_source: Corev1VolumeSource
    name: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        volume_source = self.volume_source.to_dict()

        name = self.name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "VolumeSource": volume_source,
                "name": name,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_volume_source import Corev1VolumeSource

        d = dict(src_dict)
        volume_source = Corev1VolumeSource.from_dict(d.pop("VolumeSource"))

        name = d.pop("name")

        corev_1_volume = cls(
            volume_source=volume_source,
            name=name,
        )

        corev_1_volume.additional_properties = d
        return corev_1_volume

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
