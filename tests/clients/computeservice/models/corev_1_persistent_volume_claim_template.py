from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_persistent_volume_claim_spec import (
        Corev1PersistentVolumeClaimSpec,
    )
    from ..models.metav_1_object_meta import Metav1ObjectMeta


T = TypeVar("T", bound="Corev1PersistentVolumeClaimTemplate")


@_attrs_define
class Corev1PersistentVolumeClaimTemplate:
    """
    Attributes:
        spec (Corev1PersistentVolumeClaimSpec):
        metadata (Metav1ObjectMeta | Unset):
    """

    spec: Corev1PersistentVolumeClaimSpec
    metadata: Metav1ObjectMeta | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        spec = self.spec.to_dict()

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "spec": spec,
            }
        )
        if metadata is not UNSET:
            field_dict["metadata"] = metadata

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_persistent_volume_claim_spec import (
            Corev1PersistentVolumeClaimSpec,
        )
        from ..models.metav_1_object_meta import Metav1ObjectMeta

        d = dict(src_dict)
        spec = Corev1PersistentVolumeClaimSpec.from_dict(d.pop("spec"))

        _metadata = d.pop("metadata", UNSET)
        metadata: Metav1ObjectMeta | Unset
        if isinstance(_metadata, Unset):
            metadata = UNSET
        else:
            metadata = Metav1ObjectMeta.from_dict(_metadata)

        corev_1_persistent_volume_claim_template = cls(
            spec=spec,
            metadata=metadata,
        )

        corev_1_persistent_volume_claim_template.additional_properties = d
        return corev_1_persistent_volume_claim_template

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
