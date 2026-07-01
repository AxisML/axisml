from enum import Enum


class RunPhase(str, Enum):
    CANCELING = "Canceling"
    CANCELLED = "Cancelled"
    CREATING = "Creating"
    DELETED = "Deleted"
    DELETING = "Deleting"
    FAILED = "Failed"
    PENDING = "Pending"
    RUNNING = "Running"
    SUCCEEDED = "Succeeded"

    def __str__(self) -> str:
        return str(self.value)
