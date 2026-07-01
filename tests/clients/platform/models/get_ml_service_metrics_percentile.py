from enum import Enum


class GetMLServiceMetricsPercentile(str, Enum):
    P50 = "p50"
    P95 = "p95"
    P99 = "p99"

    def __str__(self) -> str:
        return str(self.value)
