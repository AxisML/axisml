from enum import Enum


class MLServiceDesiredState(str, Enum):
    RUNNING = "Running"
    STOPPED = "Stopped"

    def __str__(self) -> str:
        return str(self.value)
