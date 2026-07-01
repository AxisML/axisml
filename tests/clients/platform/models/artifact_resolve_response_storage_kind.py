from enum import Enum


class ArtifactResolveResponseStorageKind(str, Enum):
    OCI = "oci"
    S3 = "s3"

    def __str__(self) -> str:
        return str(self.value)
