from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="Corev1ScaleIOVolumeSource")


@_attrs_define
class Corev1ScaleIOVolumeSource:
    """
    Attributes:
        gateway (str):
        system (str):
        fs_type (str | Unset):
        protection_domain (str | Unset):
        read_only (bool | Unset):
        secret_ref (Corev1LocalObjectReference | None | Unset):
        ssl_enabled (bool | Unset):
        storage_mode (str | Unset):
        storage_pool (str | Unset):
        volume_name (str | Unset):
    """

    gateway: str
    system: str
    fs_type: str | Unset = UNSET
    protection_domain: str | Unset = UNSET
    read_only: bool | Unset = UNSET
    secret_ref: Corev1LocalObjectReference | None | Unset = UNSET
    ssl_enabled: bool | Unset = UNSET
    storage_mode: str | Unset = UNSET
    storage_pool: str | Unset = UNSET
    volume_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        gateway = self.gateway

        system = self.system

        fs_type = self.fs_type

        protection_domain = self.protection_domain

        read_only = self.read_only

        secret_ref: dict[str, Any] | None | Unset
        if isinstance(self.secret_ref, Unset):
            secret_ref = UNSET
        elif isinstance(self.secret_ref, Corev1LocalObjectReference):
            secret_ref = self.secret_ref.to_dict()
        else:
            secret_ref = self.secret_ref

        ssl_enabled = self.ssl_enabled

        storage_mode = self.storage_mode

        storage_pool = self.storage_pool

        volume_name = self.volume_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "gateway": gateway,
                "system": system,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if protection_domain is not UNSET:
            field_dict["protectionDomain"] = protection_domain
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if secret_ref is not UNSET:
            field_dict["secretRef"] = secret_ref
        if ssl_enabled is not UNSET:
            field_dict["sslEnabled"] = ssl_enabled
        if storage_mode is not UNSET:
            field_dict["storageMode"] = storage_mode
        if storage_pool is not UNSET:
            field_dict["storagePool"] = storage_pool
        if volume_name is not UNSET:
            field_dict["volumeName"] = volume_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        gateway = d.pop("gateway")

        system = d.pop("system")

        fs_type = d.pop("fsType", UNSET)

        protection_domain = d.pop("protectionDomain", UNSET)

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

        ssl_enabled = d.pop("sslEnabled", UNSET)

        storage_mode = d.pop("storageMode", UNSET)

        storage_pool = d.pop("storagePool", UNSET)

        volume_name = d.pop("volumeName", UNSET)

        corev_1_scale_io_volume_source = cls(
            gateway=gateway,
            system=system,
            fs_type=fs_type,
            protection_domain=protection_domain,
            read_only=read_only,
            secret_ref=secret_ref,
            ssl_enabled=ssl_enabled,
            storage_mode=storage_mode,
            storage_pool=storage_pool,
            volume_name=volume_name,
        )

        corev_1_scale_io_volume_source.additional_properties = d
        return corev_1_scale_io_volume_source

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
