from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.oci_credentials import OciCredentials


T = TypeVar("T", bound="ArtifactResolveResponse")


@_attrs_define
class ArtifactResolveResponse:
    """
    Example:
        {'digest': 'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10', 'pullCredentials':
            {'expires_at': '2026-06-28T10:30:00Z', 'password': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.upload-token',
            'username': 'team-vision'}, 'storageKind': 'oci', 'uri': 'oci://registry.axisml.io/team-vision/resnet50:1.4.0',
            'visibility': 'tenant'}

    Attributes:
        storage_kind (str): Storage backend kind serving the artifact (e.g. oci).
        uri (str): Storage URI the client pulls the artifact content from.
        digest (str | Unset): Content digest of the resolved artifact.
        pull_credentials (None | OciCredentials | Unset): Pull credentials (carrying their own expiry), omitted for
            public artifacts that need none.
        visibility (str | Unset): Persisted visibility of the artifact (tenant or public).
    """

    storage_kind: str
    uri: str
    digest: str | Unset = UNSET
    pull_credentials: None | OciCredentials | Unset = UNSET
    visibility: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.oci_credentials import OciCredentials

        storage_kind = self.storage_kind

        uri = self.uri

        digest = self.digest

        pull_credentials: dict[str, Any] | None | Unset
        if isinstance(self.pull_credentials, Unset):
            pull_credentials = UNSET
        elif isinstance(self.pull_credentials, OciCredentials):
            pull_credentials = self.pull_credentials.to_dict()
        else:
            pull_credentials = self.pull_credentials

        visibility = self.visibility

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
        if pull_credentials is not UNSET:
            field_dict["pullCredentials"] = pull_credentials
        if visibility is not UNSET:
            field_dict["visibility"] = visibility

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.oci_credentials import OciCredentials

        d = dict(src_dict)
        storage_kind = d.pop("storageKind")

        uri = d.pop("uri")

        digest = d.pop("digest", UNSET)

        def _parse_pull_credentials(data: object) -> None | OciCredentials | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                pull_credentials_type_1 = OciCredentials.from_dict(data)

                return pull_credentials_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | OciCredentials | Unset, data)

        pull_credentials = _parse_pull_credentials(d.pop("pullCredentials", UNSET))

        visibility = d.pop("visibility", UNSET)

        artifact_resolve_response = cls(
            storage_kind=storage_kind,
            uri=uri,
            digest=digest,
            pull_credentials=pull_credentials,
            visibility=visibility,
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
