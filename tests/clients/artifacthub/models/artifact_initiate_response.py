from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.artifact import Artifact
    from ..models.upload_credentials import UploadCredentials


T = TypeVar("T", bound="ArtifactInitiateResponse")


@_attrs_define
class ArtifactInitiateResponse:
    """
    Example:
        {'artifact': {'createdAt': '2026-06-20T08:00:00Z', 'description': 'ResNet-50 image-classification model
            pretrained on ImageNet.', 'displayName': 'ResNet-50', 'id': '8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10', 'kind':
            'model', 'labels': {'stage': 'production', 'team': 'vision'}, 'name': 'resnet50', 'namespace': 'team-vision',
            'owner': 'li.wei', 'source': 'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch',
            'parameters': '25.6M', 'task': 'image-classification'}, 'status': 'Uploading', 'updatedAt':
            '2026-06-20T08:00:00Z', 'version': '1.4.0', 'visibility': 'tenant'}, 'upload': {'credentials': {'expires_at':
            '2026-06-28T10:30:00Z', 'password': 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.upload-token', 'username': 'team-
            vision'}, 'storageKind': 'oci', 'uri': 'oci://registry.axisml.io/team-vision/resnet50:1.4.0'}}

    Attributes:
        artifact (Artifact):  Example: {'annotations': {'git-commit': '8c1f4e2'}, 'createdAt': '2026-06-20T08:00:00Z',
            'description': 'ResNet-50 image-classification model pretrained on ImageNet.', 'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10', 'displayName': 'ResNet-50', 'id':
            '8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10', 'kind': 'model', 'labels': {'stage': 'production', 'team': 'vision'},
            'name': 'resnet50', 'namespace': 'team-vision', 'owner': 'li.wei', 'readyAt': '2026-06-28T09:30:00Z', 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters': '25.6M', 'task': 'image-
            classification'}, 'status': 'Ready', 'updatedAt': '2026-06-28T09:30:00Z', 'version': '1.4.0', 'visibility':
            'tenant'}.
        upload (UploadCredentials):
    """

    artifact: Artifact
    upload: UploadCredentials
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        artifact = self.artifact.to_dict()

        upload = self.upload.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "artifact": artifact,
                "upload": upload,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact import Artifact
        from ..models.upload_credentials import UploadCredentials

        d = dict(src_dict)
        artifact = Artifact.from_dict(d.pop("artifact"))

        upload = UploadCredentials.from_dict(d.pop("upload"))

        artifact_initiate_response = cls(
            artifact=artifact,
            upload=upload,
        )

        artifact_initiate_response.additional_properties = d
        return artifact_initiate_response

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
