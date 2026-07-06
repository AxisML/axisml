from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.oci_credentials import OciCredentials


T = TypeVar("T", bound="UploadCredentials")


@_attrs_define
class UploadCredentials:
    """
    Attributes:
        credentials (OciCredentials):
        storage_kind (str): Storage backend kind backing the upload (e.g. oci).
        uri (str): Target storage URI the client pushes the artifact content to.
    """

    credentials: OciCredentials
    storage_kind: str
    uri: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        credentials = self.credentials.to_dict()

        storage_kind = self.storage_kind

        uri = self.uri

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "credentials": credentials,
                "storageKind": storage_kind,
                "uri": uri,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.oci_credentials import OciCredentials

        d = dict(src_dict)
        credentials = OciCredentials.from_dict(d.pop("credentials"))

        storage_kind = d.pop("storageKind")

        uri = d.pop("uri")

        upload_credentials = cls(
            credentials=credentials,
            storage_kind=storage_kind,
            uri=uri,
        )

        upload_credentials.additional_properties = d
        return upload_credentials

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
