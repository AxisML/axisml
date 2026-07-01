from enum import Enum


class MLServicePhase(str, Enum):
    CREATING = "Creating"
    DEGRADED = "Degraded"
    DELETED = "Deleted"
    DELETING = "Deleting"
    FAILED = "Failed"
    PENDING = "Pending"
    READY = "Ready"
    STOPPED = "Stopped"

    def __str__(self) -> str:
        return str(self.value)
