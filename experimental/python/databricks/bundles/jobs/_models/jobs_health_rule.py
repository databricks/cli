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

    op: VariableOr[JobsHealthOperator]

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

    op: VariableOr[JobsHealthOperatorParam]

    value: VariableOr[int]
    """
    Specifies the threshold value that the health metric should obey to satisfy the health rule.
    """


JobsHealthRuleParam = JobsHealthRuleDict | JobsHealthRule
