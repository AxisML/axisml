from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_flex_volume_source_options import (
        Corev1FlexVolumeSourceOptions,
    )
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="Corev1FlexVolumeSource")


@_attrs_define
class Corev1FlexVolumeSource:
    """
    Attributes:
        driver (str):
        fs_type (str | Unset):
        options (Corev1FlexVolumeSourceOptions | Unset):
        read_only (bool | Unset):
        secret_ref (Corev1LocalObjectReference | None | Unset):
    """

    driver: str
    fs_type: str | Unset = UNSET
    options: Corev1FlexVolumeSourceOptions | Unset = UNSET
    read_only: bool | Unset = UNSET
    secret_ref: Corev1LocalObjectReference | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        driver = self.driver

        fs_type = self.fs_type

        options: dict[str, Any] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options.to_dict()

        read_only = self.read_only

        secret_ref: dict[str, Any] | None | Unset
        if isinstance(self.secret_ref, Unset):
            secret_ref = UNSET
        elif isinstance(self.secret_ref, Corev1LocalObjectReference):
            secret_ref = self.secret_ref.to_dict()
        else:
            secret_ref = self.secret_ref

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "driver": driver,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if options is not UNSET:
            field_dict["options"] = options
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if secret_ref is not UNSET:
            field_dict["secretRef"] = secret_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_flex_volume_source_options import (
            Corev1FlexVolumeSourceOptions,
        )
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        driver = d.pop("driver")

        fs_type = d.pop("fsType", UNSET)

        _options = d.pop("options", UNSET)
        options: Corev1FlexVolumeSourceOptions | Unset
        if isinstance(_options, Unset):
            options = UNSET
        else:
            options = Corev1FlexVolumeSourceOptions.from_dict(_options)

        read_only = d.pop("readOnly", UNSET)

        def _parse_secret_ref(
            data: object,
        ) -> Corev1LocalObjectReference | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                secret_ref_type_1 = Corev1LocalObjectReference.from_dict(data)

                return secret_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1LocalObjectReference | None | Unset, data)

        secret_ref = _parse_secret_ref(d.pop("secretRef", UNSET))

        corev_1_flex_volume_source = cls(
            driver=driver,
            fs_type=fs_type,
            options=options,
            read_only=read_only,
            secret_ref=secret_ref,
        )

        corev_1_flex_volume_source.additional_properties = d
        return corev_1_flex_volume_source

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
