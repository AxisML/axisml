from enum import Enum


class WorkloadMetricName(str, Enum):
    CPU_UTIL = "cpu_util"
    GPU_UTIL = "gpu_util"
    MEM_UTIL = "mem_util"

    def __str__(self) -> str:
        return str(self.value)
