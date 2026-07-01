from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_local_object_reference import Corev1LocalObjectReference


T = TypeVar("T", bound="Corev1ISCSIVolumeSource")


@_attrs_define
class Corev1ISCSIVolumeSource:
    """
    Attributes:
        iqn (str):
        lun (int):
        target_portal (str):
        chap_auth_discovery (bool | Unset):
        chap_auth_session (bool | Unset):
        fs_type (str | Unset):
        initiator_name (None | str | Unset):
        iscsi_interface (str | Unset):
        portals (list[str] | Unset):
        read_only (bool | Unset):
        secret_ref (Corev1LocalObjectReference | None | Unset):
    """

    iqn: str
    lun: int
    target_portal: str
    chap_auth_discovery: bool | Unset = UNSET
    chap_auth_session: bool | Unset = UNSET
    fs_type: str | Unset = UNSET
    initiator_name: None | str | Unset = UNSET
    iscsi_interface: str | Unset = UNSET
    portals: list[str] | Unset = UNSET
    read_only: bool | Unset = UNSET
    secret_ref: Corev1LocalObjectReference | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        iqn = self.iqn

        lun = self.lun

        target_portal = self.target_portal

        chap_auth_discovery = self.chap_auth_discovery

        chap_auth_session = self.chap_auth_session

        fs_type = self.fs_type

        initiator_name: None | str | Unset
        if isinstance(self.initiator_name, Unset):
            initiator_name = UNSET
        else:
            initiator_name = self.initiator_name

        iscsi_interface = self.iscsi_interface

        portals: list[str] | Unset = UNSET
        if not isinstance(self.portals, Unset):
            portals = self.portals

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
                "iqn": iqn,
                "lun": lun,
                "targetPortal": target_portal,
            }
        )
        if chap_auth_discovery is not UNSET:
            field_dict["chapAuthDiscovery"] = chap_auth_discovery
        if chap_auth_session is not UNSET:
            field_dict["chapAuthSession"] = chap_auth_session
        if fs_type is not UNSET:
            field_dict["fsType"] = fs_type
        if initiator_name is not UNSET:
            field_dict["initiatorName"] = initiator_name
        if iscsi_interface is not UNSET:
            field_dict["iscsiInterface"] = iscsi_interface
        if portals is not UNSET:
            field_dict["portals"] = portals
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if secret_ref is not UNSET:
            field_dict["secretRef"] = secret_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_local_object_reference import Corev1LocalObjectReference

        d = dict(src_dict)
        iqn = d.pop("iqn")

        lun = d.pop("lun")

        target_portal = d.pop("targetPortal")

        chap_auth_discovery = d.pop("chapAuthDiscovery", UNSET)

        chap_auth_session = d.pop("chapAuthSession", UNSET)

        fs_type = d.pop("fsType", UNSET)

        def _parse_initiator_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        initiator_name = _parse_initiator_name(d.pop("initiatorName", UNSET))

        iscsi_interface = d.pop("iscsiInterface", UNSET)

        portals = cast(list[str], d.pop("portals", UNSET))

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

        corev_1iscsi_volume_source = cls(
            iqn=iqn,
            lun=lun,
            target_portal=target_portal,
            chap_auth_discovery=chap_auth_discovery,
            chap_auth_session=chap_auth_session,
            fs_type=fs_type,
            initiator_name=initiator_name,
            iscsi_interface=iscsi_interface,
            portals=portals,
            read_only=read_only,
            secret_ref=secret_ref,
        )

        corev_1iscsi_volume_source.additional_properties = d
        return corev_1iscsi_volume_source

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
