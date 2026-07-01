from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="Corev1RBDVolumeSource")


@_attrs_define
class Corev1RBDVolumeSource:
    """
    Attributes:
        image (str):
        monitors (list[str]):
        fs_type (str | Unset):
        keyring (str | Unset):
        pool (str | Unset):
        read_only (bool | Unset):
        secret_ref (Corev1LocalObjectReference | None | Unset):
        user (str | Unset):
    """

    image: str
    monitors: list[str]
    fs_type: str | Unset = UNSET
    keyring: str | Unset = UNSET
    pool: str | Unset = UNSET
    read_only: bool | Unset = UNSET
    secret_ref: Corev1LocalObjectReference | None | Unset = UNSET
    user: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        image = self.image

        monitors = self.monitors

        fs_type = self.fs_type

        keyring = self.keyring

        pool = self.pool

        read_only = self.read_only

        secret_ref: dict[str, Any] | None | Unset
        if isinstance(self.secret_ref, Unset):
            secret_ref = UNSET
        elif isinstance(self.secret_ref, Corev1LocalObjectReference):
            secret_ref = self.secret_ref.to_dict()
        else:
            secret_ref = self.secret_ref

        user = self.user

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "image": image,
                "monitors": monitors,
            }
        )
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if keyring is not UNSET:
            field_dict["keyring"] = keyring
        if pool is not UNSET:
            field_dict["pool"] = pool
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if secret_ref is not UNSET:
            field_dict["secretRef"] = secret_ref
        if user is not UNSET:
            field_dict["user"] = user

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        image = d.pop("image")

        monitors = cast(list[str], d.pop("monitors"))

        fs_type = d.pop("fsType", UNSET)

        keyring = d.pop("keyring", UNSET)

        pool = d.pop("pool", UNSET)

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

        user = d.pop("user", UNSET)

        corev_1rbd_volume_source = cls(
            image=image,
            monitors=monitors,
            fs_type=fs_type,
            keyring=keyring,
            pool=pool,
            read_only=read_only,
            secret_ref=secret_ref,
            user=user,
        )

        corev_1rbd_volume_source.additional_properties = d
        return corev_1rbd_volume_source

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
