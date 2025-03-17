from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.cluster_spec import (
    ClusterSpec,
    ClusterSpecParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobCluster:
    """"""

    job_cluster_key: VariableOr[str]
    """
    A unique name for the job cluster. This field is required and must be unique within the job.
    `JobTaskSettings` may refer to this field to determine which cluster to launch for the task execution.
    """

    new_cluster: VariableOr[ClusterSpec]
    """
    If new_cluster, a description of a cluster that is created for each task.
    """

    @classmethod
    def from_dict(cls, value: "JobClusterDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobClusterDict":
        return _transform_to_json_value(self)  # type:ignore


class JobClusterDict(TypedDict, total=False):
    """"""

    job_cluster_key: VariableOr[str]
    """
    A unique name for the job cluster. This field is required and must be unique within the job.
    `JobTaskSettings` may refer to this field to determine which cluster to launch for the task execution.
    """

    new_cluster: VariableOr[ClusterSpecParam]
    """
    If new_cluster, a description of a cluster that is created for each task.
    """


JobClusterParam = JobClusterDict | JobCluster
