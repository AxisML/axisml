from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.image_initiate_response_storage_kind import (
    ImageInitiateResponseStorageKind,
)
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.image_initiate_response_upload_credentials import (
        ImageInitiateResponseUploadCredentials,
    )


T = TypeVar("T", bound="ImageInitiateResponse")


@_attrs_define
class ImageInitiateResponse:
    """
    Example:
        {'expiresAt': '2026-06-28T09:30:00Z', 'id': 'a1b2c3d4-5e6f-7081-92a3-b4c5d6e7f809', 'storageKind': 'oci',
            'uploadCredentials': {'password': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9', 'username': 'upload-token'}, 'uri':
            'oci://registry.axisml.io/team-vision/pytorch-train:2.3.0'}

    Attributes:
        id (UUID): Identifier of the newly initiated image version.
        uri (str): Push target URI for the image content.
        expires_at (datetime.datetime | Unset): Expiry time of the upload credentials.
        storage_kind (ImageInitiateResponseStorageKind | Unset): Backing store of the push target (oci, s3).
        upload_credentials (ImageInitiateResponseUploadCredentials | Unset): Short-lived credentials for pushing the
            image.
    """

    id: UUID
    uri: str
    expires_at: datetime.datetime | Unset = UNSET
    storage_kind: ImageInitiateResponseStorageKind | Unset = UNSET
    upload_credentials: ImageInitiateResponseUploadCredentials | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = str(self.id)

        uri = self.uri

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        storage_kind: str | Unset = UNSET
        if not isinstance(self.storage_kind, Unset):
            storage_kind = self.storage_kind.value

        upload_credentials: dict[str, Any] | Unset = UNSET
        if not isinstance(self.upload_credentials, Unset):
            upload_credentials = self.upload_credentials.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "uri": uri,
            }
        )
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if storage_kind is not UNSET:
            field_dict["storageKind"] = storage_kind
        if upload_credentials is not UNSET:
            field_dict["uploadCredentials"] = upload_credentials

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.image_initiate_response_upload_credentials import (
            ImageInitiateResponseUploadCredentials,
        )

        d = dict(src_dict)
        id = UUID(d.pop("id"))

        uri = d.pop("uri")

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at, Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)

        _storage_kind = d.pop("storageKind", UNSET)
        storage_kind: ImageInitiateResponseStorageKind | Unset
        if isinstance(_storage_kind, Unset):
            storage_kind = UNSET
        else:
            storage_kind = ImageInitiateResponseStorageKind(_storage_kind)

        _upload_credentials = d.pop("uploadCredentials", UNSET)
        upload_credentials: ImageInitiateResponseUploadCredentials | Unset
        if isinstance(_upload_credentials, Unset):
            upload_credentials = UNSET
        else:
            upload_credentials = ImageInitiateResponseUploadCredentials.from_dict(
                _upload_credentials
            )

        image_initiate_response = cls(
            id=id,
            uri=uri,
            expires_at=expires_at,
            storage_kind=storage_kind,
            upload_credentials=upload_credentials,
        )

        image_initiate_response.additional_properties = d
        return image_initiate_response

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
