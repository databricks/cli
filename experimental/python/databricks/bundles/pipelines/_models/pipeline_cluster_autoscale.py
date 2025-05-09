from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.pipelines._models.pipeline_cluster_autoscale_mode import (
    PipelineClusterAutoscaleMode,
    PipelineClusterAutoscaleModeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineClusterAutoscale:
    """"""

    max_workers: VariableOr[int]
    """
    The maximum number of workers to which the cluster can scale up when overloaded. `max_workers` must be strictly greater than `min_workers`.
    """

    min_workers: VariableOr[int]
    """
    The minimum number of workers the cluster can scale down to when underutilized.
    It is also the initial number of workers the cluster will have after creation.
    """

    mode: VariableOrOptional[PipelineClusterAutoscaleMode] = None
    """
    Databricks Enhanced Autoscaling optimizes cluster utilization by automatically
    allocating cluster resources based on workload volume, with minimal impact to
    the data processing latency of your pipelines. Enhanced Autoscaling is available
    for `updates` clusters only. The legacy autoscaling feature is used for `maintenance`
    clusters.
    """

    @classmethod
    def from_dict(cls, value: "PipelineClusterAutoscaleDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineClusterAutoscaleDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineClusterAutoscaleDict(TypedDict, total=False):
    """"""

    max_workers: VariableOr[int]
    """
    The maximum number of workers to which the cluster can scale up when overloaded. `max_workers` must be strictly greater than `min_workers`.
    """

    min_workers: VariableOr[int]
    """
    The minimum number of workers the cluster can scale down to when underutilized.
    It is also the initial number of workers the cluster will have after creation.
    """

    mode: VariableOrOptional[PipelineClusterAutoscaleModeParam]
    """
    Databricks Enhanced Autoscaling optimizes cluster utilization by automatically
    allocating cluster resources based on workload volume, with minimal impact to
    the data processing latency of your pipelines. Enhanced Autoscaling is available
    for `updates` clusters only. The legacy autoscaling feature is used for `maintenance`
    clusters.
    """


PipelineClusterAutoscaleParam = PipelineClusterAutoscaleDict | PipelineClusterAutoscale
