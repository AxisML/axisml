from enum import Enum


class EventType(str, Enum):
    NORMAL = "Normal"
    WARNING = "Warning"

    def __str__(self) -> str:
        return str(self.value)
