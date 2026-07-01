from enum import Enum


class ImageStatus(str, Enum):
    DELETED = "Deleted"
    DELETING = "Deleting"
    FAILED = "Failed"
    READY = "Ready"
    UPLOADING = "Uploading"

    def __str__(self) -> str:
        return str(self.value)
