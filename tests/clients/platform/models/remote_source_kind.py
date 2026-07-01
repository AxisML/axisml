from enum import Enum


class RemoteSourceKind(str, Enum):
    CUSTOM = "custom"
    HF = "hf"
    HTTP = "http"
    OCI = "oci"
    S3 = "s3"

    def __str__(self) -> str:
        return str(self.value)
