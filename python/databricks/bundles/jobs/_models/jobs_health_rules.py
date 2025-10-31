from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList
from databricks.bundles.jobs._models.jobs_health_rule import (
    JobsHealthRule,
    JobsHealthRuleParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobsHealthRules:
    """
    An optional set of health rules that can be defined for this job.
    """

    rules: VariableOrList[JobsHealthRule] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "JobsHealthRulesDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobsHealthRulesDict":
        return _transform_to_json_value(self)  # type:ignore


class JobsHealthRulesDict(TypedDict, total=False):
    """"""

    rules: VariableOrList[JobsHealthRuleParam]


JobsHealthRulesParam = JobsHealthRulesDict | JobsHealthRules
