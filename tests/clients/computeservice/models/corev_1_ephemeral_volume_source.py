from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_persistent_volume_claim_template import (
        Corev1PersistentVolumeClaimTemplate,
    )


T = TypeVar("T", bound="Corev1EphemeralVolumeSource")


@_attrs_define
class Corev1EphemeralVolumeSource:
    """
    Attributes:
        volume_claim_template (Corev1PersistentVolumeClaimTemplate | None | Unset):
    """

    volume_claim_template: Corev1PersistentVolumeClaimTemplate | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_persistent_volume_claim_template import (
            Corev1PersistentVolumeClaimTemplate,
        )

        volume_claim_template: dict[str, Any] | None | Unset
        if isinstance(self.volume_claim_template, Unset):
            volume_claim_template = UNSET
        elif isinstance(
            self.volume_claim_template, Corev1PersistentVolumeClaimTemplate
        ):
            volume_claim_template = self.volume_claim_template.to_dict()
        else:
            volume_claim_template = self.volume_claim_template

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if volume_claim_template is not UNSET:
            field_dict["volumeClaimTemplate"] = volume_claim_template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_persistent_volume_claim_template import (
            Corev1PersistentVolumeClaimTemplate,
        )

        d = dict(src_dict)

        def _parse_volume_claim_template(
            data: object,
        ) -> Corev1PersistentVolumeClaimTemplate | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                volume_claim_template_type_1 = (
                    Corev1PersistentVolumeClaimTemplate.from_dict(data)
                )

                return volume_claim_template_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1PersistentVolumeClaimTemplate | None | Unset, data)

        volume_claim_template = _parse_volume_claim_template(
            d.pop("volumeClaimTemplate", UNSET)
        )

        corev_1_ephemeral_volume_source = cls(
            volume_claim_template=volume_claim_template,
        )

        corev_1_ephemeral_volume_source.additional_properties = d
        return corev_1_ephemeral_volume_source

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
