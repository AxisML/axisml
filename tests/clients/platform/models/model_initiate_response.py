from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.model_initiate_response_storage_kind import (
    ModelInitiateResponseStorageKind,
)
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.model_initiate_response_upload_credentials import (
        ModelInitiateResponseUploadCredentials,
    )


T = TypeVar("T", bound="ModelInitiateResponse")


@_attrs_define
class ModelInitiateResponse:
    """
    Example:
        {'expiresAt': '2026-06-28T09:30:00Z', 'id': 'c4a7f1e9-3d2b-4a6c-8e1f-9b0d5a2c7f31', 'storageKind': 'oci',
            'uploadCredentials': {'password': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9', 'username': 'upload-token'}, 'uri':
            'oci://registry.axisml.io/team-vision/resnet50:1.4.0'}

    Attributes:
        id (UUID): Identifier of the newly initiated model version.
        uri (str): Upload target URI for the version content.
        expires_at (datetime.datetime | Unset): Expiry time of the upload credentials.
        storage_kind (ModelInitiateResponseStorageKind | Unset): Backing store of the upload target (oci, s3).
        upload_credentials (ModelInitiateResponseUploadCredentials | Unset): Short-lived credentials for pushing the
            content.
    """

    id: UUID
    uri: str
    expires_at: datetime.datetime | Unset = UNSET
    storage_kind: ModelInitiateResponseStorageKind | Unset = UNSET
    upload_credentials: ModelInitiateResponseUploadCredentials | Unset = UNSET
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
        from ..models.model_initiate_response_upload_credentials import (
            ModelInitiateResponseUploadCredentials,
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
        storage_kind: ModelInitiateResponseStorageKind | Unset
        if isinstance(_storage_kind, Unset):
            storage_kind = UNSET
        else:
            storage_kind = ModelInitiateResponseStorageKind(_storage_kind)

        _upload_credentials = d.pop("uploadCredentials", UNSET)
        upload_credentials: ModelInitiateResponseUploadCredentials | Unset
        if isinstance(_upload_credentials, Unset):
            upload_credentials = UNSET
        else:
            upload_credentials = ModelInitiateResponseUploadCredentials.from_dict(
                _upload_credentials
            )

        model_initiate_response = cls(
            id=id,
            uri=uri,
            expires_at=expires_at,
            storage_kind=storage_kind,
            upload_credentials=upload_credentials,
        )

        model_initiate_response.additional_properties = d
        return model_initiate_response

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
