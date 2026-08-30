from enum import Enum


class WorkspacePhase(str, Enum):
    CREATING = "Creating"
    DEGRADED = "Degraded"
    DELETED = "Deleted"
    DELETING = "Deleting"
    FAILED = "Failed"
    PENDING = "Pending"
    QUEUED = "Queued"
    RUNNING = "Running"
    STARTING = "Starting"
    STOPPED = "Stopped"

    def __str__(self) -> str:
        return str(self.value)
