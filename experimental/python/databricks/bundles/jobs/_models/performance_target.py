from enum import Enum
from typing import Literal


class PerformanceTarget(Enum):
    """
    :meta private: [EXPERIMENTAL]

    PerformanceTarget defines how performant (lower latency) or cost efficient the execution of run on serverless compute should be.
    The performance mode on the job or pipeline should map to a performance setting that is passed to Cluster Manager
    (see cluster-common PerformanceTarget).
    """

    PERFORMANCE_OPTIMIZED = "PERFORMANCE_OPTIMIZED"
    COST_OPTIMIZED = "COST_OPTIMIZED"
    BALANCED = "BALANCED"


PerformanceTargetParam = (
    Literal["PERFORMANCE_OPTIMIZED", "COST_OPTIMIZED", "BALANCED"] | PerformanceTarget
)
