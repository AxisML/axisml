from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_volume_projection import Corev1VolumeProjection


T = TypeVar("T", bound="Corev1ProjectedVolumeSource")


@_attrs_define
class Corev1ProjectedVolumeSource:
    """
    Attributes:
        sources (list[Corev1VolumeProjection]):
        default_mode (int | None | Unset):
    """

    sources: list[Corev1VolumeProjection]
    default_mode: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)

        default_mode: int | None | Unset
        if isinstance(self.default_mode, Unset):
            default_mode = UNSET
        else:
            default_mode = self.default_mode

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "sources": sources,
            }
        )
        if default_mode is not UNSET:
            field_dict["defaultMode"] = default_mode

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_volume_projection import Corev1VolumeProjection

        d = dict(src_dict)
        sources = []
        _sources = d.pop("sources")
        for sources_item_data in _sources:
            sources_item = Corev1VolumeProjection.from_dict(sources_item_data)

            sources.append(sources_item)

        def _parse_default_mode(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        default_mode = _parse_default_mode(d.pop("defaultMode", UNSET))

        corev_1_projected_volume_source = cls(
            sources=sources,
            default_mode=default_mode,
        )

        corev_1_projected_volume_source.additional_properties = d
        return corev_1_projected_volume_source

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
