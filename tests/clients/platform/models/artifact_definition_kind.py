from enum import Enum


class ArtifactDefinitionKind(str, Enum):
    IMAGE = "image"
    MODEL = "model"

    def __str__(self) -> str:
        return str(self.value)
