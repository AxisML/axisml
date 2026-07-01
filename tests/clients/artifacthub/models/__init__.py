"""Contains all the data models used in inputs/outputs"""

from .artifact import Artifact
from .artifact_annotations import ArtifactAnnotations
from .artifact_complete_request import ArtifactCompleteRequest
from .artifact_hub_error import ArtifactHubError
from .artifact_hub_error_code import ArtifactHubErrorCode
from .artifact_hub_error_details import ArtifactHubErrorDetails
from .artifact_initiate_request import ArtifactInitiateRequest
from .artifact_initiate_request_annotations import ArtifactInitiateRequestAnnotations
from .artifact_initiate_request_labels import ArtifactInitiateRequestLabels
from .artifact_initiate_request_spec import ArtifactInitiateRequestSpec
from .artifact_initiate_response import ArtifactInitiateResponse
from .artifact_labels import ArtifactLabels
from .artifact_list import ArtifactList
from .artifact_patch_request import ArtifactPatchRequest
from .artifact_patch_request_annotations import ArtifactPatchRequestAnnotations
from .artifact_patch_request_labels import ArtifactPatchRequestLabels
from .artifact_resolve_response import ArtifactResolveResponse
from .artifact_spec import ArtifactSpec
from .capabilities import Capabilities
from .oci_credentials import OciCredentials
from .upload_credentials import UploadCredentials

__all__ = (
    "Artifact",
    "ArtifactAnnotations",
    "ArtifactCompleteRequest",
    "ArtifactHubError",
    "ArtifactHubErrorCode",
    "ArtifactHubErrorDetails",
    "ArtifactInitiateRequest",
    "ArtifactInitiateRequestAnnotations",
    "ArtifactInitiateRequestLabels",
    "ArtifactInitiateRequestSpec",
    "ArtifactInitiateResponse",
    "ArtifactLabels",
    "ArtifactList",
    "ArtifactPatchRequest",
    "ArtifactPatchRequestAnnotations",
    "ArtifactPatchRequestLabels",
    "ArtifactResolveResponse",
    "ArtifactSpec",
    "Capabilities",
    "OciCredentials",
    "UploadCredentials",
)
