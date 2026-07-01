from enum import Enum


class TrafficPolicyMode(str, Enum):
    CANARY = "canary"
    WEIGHTED = "weighted"

    def __str__(self) -> str:
        return str(self.value)
