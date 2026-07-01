from enum import Enum


class ImagePurpose(str, Enum):
    CUSTOM = "custom"
    INFERENCE = "inference"
    TRAINING = "training"
    WORKSPACE = "workspace"

    def __str__(self) -> str:
        return str(self.value)
