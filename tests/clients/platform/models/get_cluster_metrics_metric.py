from enum import Enum


class GetClusterMetricsMetric(str, Enum):
    CPU_UTIL = "cpu_util"
    GPU_QUOTA = "gpu_quota"
    GPU_UTIL = "gpu_util"
    MEM_UTIL = "mem_util"

    def __str__(self) -> str:
        return str(self.value)
