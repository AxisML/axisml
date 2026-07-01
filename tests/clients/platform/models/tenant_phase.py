from enum import Enum


class TenantPhase(str, Enum):
    ACTIVE = "Active"
    CREATING = "Creating"
    FAILED = "Failed"
    SUSPENDED = "Suspended"

    def __str__(self) -> str:
        return str(self.value)
