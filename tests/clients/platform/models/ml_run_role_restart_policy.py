from enum import Enum


class MLRunRoleRestartPolicy(str, Enum):
    ALWAYS = "Always"
    NEVER = "Never"
    ONFAILURE = "OnFailure"

    def __str__(self) -> str:
        return str(self.value)
