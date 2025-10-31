from enum import Enum
from typing import Literal


class PipelineClusterAutoscaleMode(Enum):
    """
    Databricks Enhanced Autoscaling optimizes cluster utilization by automatically
    allocating cluster resources based on workload volume, with minimal impact to
    the data processing latency of your pipelines. Enhanced Autoscaling is available
    for `updates` clusters only. The legacy autoscaling feature is used for `maintenance`
    clusters.
    """

    ENHANCED = "ENHANCED"
    LEGACY = "LEGACY"


PipelineClusterAutoscaleModeParam = (
    Literal["ENHANCED", "LEGACY"] | PipelineClusterAutoscaleMode
)
