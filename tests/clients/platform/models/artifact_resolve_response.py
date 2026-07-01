from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.artifact_resolve_response_storage_kind import (
    ArtifactResolveResponseStorageKind,
)
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_resolve_response_pull_credentials import (
        ArtifactResolveResponsePullCredentials,
    )


T = TypeVar("T", bound="ArtifactResolveResponse")


@_attrs_define
class ArtifactResolveResponse:
    """
    Example:
        {'digest': 'sha256:9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a', 'expiresAt':
            '2026-06-28T09:30:00Z', 'pullCredentials': {'password': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9', 'username':
            'pull-token'}, 'storageKind': 'oci', 'uri': 'oci://registry.axisml.io/team-vision/resnet50:1.4.0'}

    Attributes:
        storage_kind (ArtifactResolveResponseStorageKind): Backing store of the download target (oci, s3).
        uri (str): Download URI for the version content.
        digest (str | Unset): Content digest of the version.
        expires_at (datetime.datetime | Unset): Expiry time of the pull credentials.
        pull_credentials (ArtifactResolveResponsePullCredentials | Unset): Short-lived credentials for pulling the
            content.
    """

    storage_kind: ArtifactResolveResponseStorageKind
    uri: str
    digest: str | Unset = UNSET
    expires_at: datetime.datetime | Unset = UNSET
    pull_credentials: ArtifactResolveResponsePullCredentials | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        storage_kind = self.storage_kind.value

        uri = self.uri

        digest = self.digest

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        pull_credentials: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pull_credentials, Unset):
            pull_credentials = self.pull_credentials.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "storageKind": storage_kind,
                "uri": uri,
            }
        )
        if digest is not UNSET:
            field_dict["digest"] = digest
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if pull_credentials is not UNSET:
            field_dict["pullCredentials"] = pull_credentials

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_resolve_response_pull_credentials import (
            ArtifactResolveResponsePullCredentials,
        )

        d = dict(src_dict)
        storage_kind = ArtifactResolveResponseStorageKind(d.pop("storageKind"))

        uri = d.pop("uri")

        digest = d.pop("digest", UNSET)

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at, Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)

        _pull_credentials = d.pop("pullCredentials", UNSET)
        pull_credentials: ArtifactResolveResponsePullCredentials | Unset
        if isinstance(_pull_credentials, Unset):
            pull_credentials = UNSET
        else:
            pull_credentials = ArtifactResolveResponsePullCredentials.from_dict(
                _pull_credentials
            )

        artifact_resolve_response = cls(
            storage_kind=storage_kind,
            uri=uri,
            digest=digest,
            expires_at=expires_at,
            pull_credentials=pull_credentials,
        )

        artifact_resolve_response.additional_properties = d
        return artifact_resolve_response

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
