from enum import Enum


class ConditionStatus(str, Enum):
    FALSE = "False"
    TRUE = "True"
    UNKNOWN = "Unknown"

    def __str__(self) -> str:
        return str(self.value)
