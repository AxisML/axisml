from enum import Enum


class MemberCreateRequestRoleName(str, Enum):
    TENANT_ADMIN = "tenant-admin"
    USER = "user"

    def __str__(self) -> str:
        return str(self.value)
