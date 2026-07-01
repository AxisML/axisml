from enum import Enum


class BackendName(str, Enum):
    CUSTOM = "custom"
    KSERVE = "kserve"
    KUBEFLOW_TRAINER = "kubeflow-trainer"
    NATIVE = "native"

    def __str__(self) -> str:
        return str(self.value)
