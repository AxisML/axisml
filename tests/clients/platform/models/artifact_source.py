from enum import Enum


class ArtifactSource(str, Enum):
    DOCKERPUSH = "dockerPush"
    EXTERNAL = "external"
    ORAS = "oras"
    WEBUPLOAD = "webUpload"

    def __str__(self) -> str:
        return str(self.value)
