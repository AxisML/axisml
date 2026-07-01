from enum import Enum


class MLServiceMetricName(str, Enum):
    CPU_UTIL = "cpu_util"
    ERROR_RATE = "error_rate"
    GPU_UTIL = "gpu_util"
    LATENCY = "latency"
    MEM_UTIL = "mem_util"
    REQUEST_RATE = "request_rate"

    def __str__(self) -> str:
        return str(self.value)
