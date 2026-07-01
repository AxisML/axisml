from enum import Enum


class ArtifactHubErrorCode(str, Enum):
    CONFLICT = "conflict"
    FORBIDDEN = "forbidden"
    GONE = "gone"
    INTERNAL_ERROR = "internal_error"
    NOT_FOUND = "not_found"
    PRECONDITION_FAILED = "precondition_failed"
    SERVICE_UNAVAILABLE = "service_unavailable"
    UNAUTHORIZED = "unauthorized"
    VALIDATION_FAILED = "validation_failed"

    def __str__(self) -> str:
        return str(self.value)
