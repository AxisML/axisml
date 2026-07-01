from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference
    from ..models.corev_1csi_volume_source_volume_attributes import (
        Corev1CSIVolumeSourceVolumeAttributes,
    )


T = TypeVar("T", bound="Corev1CSIVolumeSource")


@_attrs_define
class Corev1CSIVolumeSource:
    """
    Attributes:
        driver (str):
        fs_type (None | str | Unset):
        node_publish_secret_ref (Corev1LocalObjectReference | None | Unset):
        read_only (bool | None | Unset):
        volume_attributes (Corev1CSIVolumeSourceVolumeAttributes | Unset):
    """

    driver: str
    fs_type: None | str | Unset = UNSET
    node_publish_secret_ref: Corev1LocalObjectReference | None | Unset = UNSET
    read_only: bool | None | Unset = UNSET
    volume_attributes: Corev1CSIVolumeSourceVolumeAttributes | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        driver = self.driver

        fs_type: None | str | Unset
        if isinstance(self.fs_type, Unset):
            fs_type = UNSET
        else:
            fs_type = self.fs_type

        node_publish_secret_ref: dict[str, Any] | None | Unset
        if isinstance(self.node_publish_secret_ref, Unset):
            node_publish_secret_ref = UNSET
        elif isinstance(self.node_publish_secret_ref, Corev1LocalObjectReference):
            node_publish_secret_ref = self.node_publish_secret_ref.to_dict()
        else:
            node_publish_secret_ref = self.node_publish_secret_ref

        read_only: bool | None | Unset
        if isinstance(self.read_only, Unset):
            read_only = UNSET
        else:
            read_only = self.read_only

        volume_attributes: dict[str, Any] | Unset = UNSET
        if not isinstance(self.volume_attributes, Unset):
            volume_attributes = self.volume_attributes.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "driver": driver,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if node_publish_secret_ref is not UNSET:
            field_dict["nodePublishSecretRef"] = node_publish_secret_ref
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if volume_attributes is not UNSET:
            field_dict["volumeAttributes"] = volume_attributes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference
        from ..models.corev_1csi_volume_source_volume_attributes import (
            Corev1CSIVolumeSourceVolumeAttributes,
        )

        d = dict(src_dict)
        driver = d.pop("driver")

        def _parse_fs_type(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        fs_type = _parse_fs_type(d.pop("fsType", UNSET))

        def _parse_node_publish_secret_ref(
            data: object,
        ) -> Corev1LocalObjectReference | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                node_publish_secret_ref_type_1 = Corev1LocalObjectReference.from_dict(
                    data
                )

                return node_publish_secret_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1LocalObjectReference | None | Unset, data)

        node_publish_secret_ref = _parse_node_publish_secret_ref(
            d.pop("nodePublishSecretRef", UNSET)
        )

        def _parse_read_only(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        read_only = _parse_read_only(d.pop("readOnly", UNSET))

        _volume_attributes = d.pop("volumeAttributes", UNSET)
        volume_attributes: Corev1CSIVolumeSourceVolumeAttributes | Unset
        if isinstance(_volume_attributes, Unset):
            volume_attributes = UNSET
        else:
            volume_attributes = Corev1CSIVolumeSourceVolumeAttributes.from_dict(
                _volume_attributes
            )

        corev_1csi_volume_source = cls(
            driver=driver,
            fs_type=fs_type,
            node_publish_secret_ref=node_publish_secret_ref,
            read_only=read_only,
            volume_attributes=volume_attributes,
        )

        corev_1csi_volume_source.additional_properties = d
        return corev_1csi_volume_source

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
