from enum import Enum


class RoleName(str, Enum):
    SYSTEM_ADMIN = "system-admin"
    TENANT_ADMIN = "tenant-admin"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)
