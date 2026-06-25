from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr
from databricks.bundles.jobs._models.jobs_health_metric import (
    JobsHealthMetric,
    JobsHealthMetricParam,
)
from databricks.bundles.jobs._models.jobs_health_operator import (
    JobsHealthOperator,
    JobsHealthOperatorParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobsHealthRule:
    """"""

    metric: VariableOr[JobsHealthMetric]
    """
    Specifies the health metric that is being evaluated for a particular health rule.
    
    * `RUN_DURATION_SECONDS`: Expected total time for a run in seconds.
    * `STREAMING_BACKLOG_BYTES`: An estimate of the maximum bytes of data waiting to be consumed across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_RECORDS`: An estimate of the maximum offset lag across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_SECONDS`: An estimate of the maximum consumer delay across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_FILES`: An estimate of the maximum number of outstanding files across all streams. This metric is in Public Preview.
    """

    op: VariableOr[JobsHealthOperator]
    """
    Specifies the operator used to compare the health metric value with the specified threshold.
    """

    value: VariableOr[int]
    """
    Specifies the threshold value that the health metric should obey to satisfy the health rule.
    """

    @classmethod
    def from_dict(cls, value: "JobsHealthRuleDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobsHealthRuleDict":
        return _transform_to_json_value(self)  # type:ignore


class JobsHealthRuleDict(TypedDict, total=False):
    """"""

    metric: VariableOr[JobsHealthMetricParam]
    """
    Specifies the health metric that is being evaluated for a particular health rule.
    
    * `RUN_DURATION_SECONDS`: Expected total time for a run in seconds.
    * `STREAMING_BACKLOG_BYTES`: An estimate of the maximum bytes of data waiting to be consumed across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_RECORDS`: An estimate of the maximum offset lag across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_SECONDS`: An estimate of the maximum consumer delay across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_FILES`: An estimate of the maximum number of outstanding files across all streams. This metric is in Public Preview.
    """

    op: VariableOr[JobsHealthOperatorParam]
    """
    Specifies the operator used to compare the health metric value with the specified threshold.
    """

    value: VariableOr[int]
    """
    Specifies the threshold value that the health metric should obey to satisfy the health rule.
    """


JobsHealthRuleParam = JobsHealthRuleDict | JobsHealthRule
