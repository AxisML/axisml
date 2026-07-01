from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_pod_certificate_projection_user_annotations import (
        Corev1PodCertificateProjectionUserAnnotations,
    )


T = TypeVar("T", bound="Corev1PodCertificateProjection")


@_attrs_define
class Corev1PodCertificateProjection:
    """
    Attributes:
        certificate_chain_path (str | Unset):
        credential_bundle_path (str | Unset):
        key_path (str | Unset):
        key_type (str | Unset):
        max_expiration_seconds (int | None | Unset):
        signer_name (str | Unset):
        user_annotations (Corev1PodCertificateProjectionUserAnnotations | Unset):
    """

    certificate_chain_path: str | Unset = UNSET
    credential_bundle_path: str | Unset = UNSET
    key_path: str | Unset = UNSET
    key_type: str | Unset = UNSET
    max_expiration_seconds: int | None | Unset = UNSET
    signer_name: str | Unset = UNSET
    user_annotations: Corev1PodCertificateProjectionUserAnnotations | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        certificate_chain_path = self.certificate_chain_path

        credential_bundle_path = self.credential_bundle_path

        key_path = self.key_path

        key_type = self.key_type

        max_expiration_seconds: int | None | Unset
        if isinstance(self.max_expiration_seconds, Unset):
            max_expiration_seconds = UNSET
        else:
            max_expiration_seconds = self.max_expiration_seconds

        signer_name = self.signer_name

        user_annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.user_annotations, Unset):
            user_annotations = self.user_annotations.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if certificate_chain_path is not UNSET:
            field_dict["certificateChainPath"] = certificate_chain_path
        if credential_bundle_path is not UNSET:
            field_dict["credentialBundlePath"] = credential_bundle_path
        if key_path is not UNSET:
            field_dict["keyPath"] = key_path
        if key_type is not UNSET:
            field_dict["keyType"] = key_type
        if max_expiration_seconds is not UNSET:
            field_dict["maxExpirationSeconds"] = max_expiration_seconds
        if signer_name is not UNSET:
            field_dict["signerName"] = signer_name
        if user_annotations is not UNSET:
            field_dict["userAnnotations"] = user_annotations

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_pod_certificate_projection_user_annotations import (
            Corev1PodCertificateProjectionUserAnnotations,
        )

        d = dict(src_dict)
        certificate_chain_path = d.pop("certificateChainPath", UNSET)

        credential_bundle_path = d.pop("credentialBundlePath", UNSET)

        key_path = d.pop("keyPath", UNSET)

        key_type = d.pop("keyType", UNSET)

        def _parse_max_expiration_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        max_expiration_seconds = _parse_max_expiration_seconds(
            d.pop("maxExpirationSeconds", UNSET)
        )

        signer_name = d.pop("signerName", UNSET)

        _user_annotations = d.pop("userAnnotations", UNSET)
        user_annotations: Corev1PodCertificateProjectionUserAnnotations | Unset
        if isinstance(_user_annotations, Unset):
            user_annotations = UNSET
        else:
            user_annotations = Corev1PodCertificateProjectionUserAnnotations.from_dict(
                _user_annotations
            )

        corev_1_pod_certificate_projection = cls(
            certificate_chain_path=certificate_chain_path,
            credential_bundle_path=credential_bundle_path,
            key_path=key_path,
            key_type=key_type,
            max_expiration_seconds=max_expiration_seconds,
            signer_name=signer_name,
            user_annotations=user_annotations,
        )

        corev_1_pod_certificate_projection.additional_properties = d
        return corev_1_pod_certificate_projection

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
