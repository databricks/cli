from enum import Enum
from typing import Literal


class PerformanceTarget(Enum):
    """
    PerformanceTarget defines how performant (lower latency) or cost efficient the execution of run on serverless compute should be.
    The performance mode on the job or pipeline should map to a performance setting that is passed to Cluster Manager
    (see cluster-common PerformanceTarget).
    """

    PERFORMANCE_OPTIMIZED = "PERFORMANCE_OPTIMIZED"
    STANDARD = "STANDARD"


PerformanceTargetParam = (
    Literal["PERFORMANCE_OPTIMIZED", "STANDARD"] | PerformanceTarget
)
