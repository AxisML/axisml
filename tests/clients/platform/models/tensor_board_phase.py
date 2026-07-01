from enum import Enum


class TensorBoardPhase(str, Enum):
    FAILED = "Failed"
    PENDING = "Pending"
    READY = "Ready"
    STOPPED = "Stopped"

    def __str__(self) -> str:
        return str(self.value)
