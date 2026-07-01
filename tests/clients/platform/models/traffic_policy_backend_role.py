from enum import Enum


class TrafficPolicyBackendRole(str, Enum):
    CANARY = "canary"
    MEMBER = "member"
    STABLE = "stable"

    def __str__(self) -> str:
        return str(self.value)
