from enum import Enum


class TrafficPolicyPhase(str, Enum):
    CREATING = "Creating"
    DEGRADED = "Degraded"
    DELETED = "Deleted"
    DELETING = "Deleting"
    FAILED = "Failed"
    PENDING = "Pending"
    READY = "Ready"

    def __str__(self) -> str:
        return str(self.value)
