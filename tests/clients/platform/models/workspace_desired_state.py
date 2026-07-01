from enum import Enum


class WorkspaceDesiredState(str, Enum):
    RUNNING = "Running"
    STOPPED = "Stopped"

    def __str__(self) -> str:
        return str(self.value)
