from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr
from databricks.bundles.jobs._models.condition_task_op import (
    ConditionTaskOp,
    ConditionTaskOpParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ConditionTask:
    """"""

    left: VariableOr[str]
    """
    The left operand of the condition task. Can be either a string value or a job state or parameter reference.
    """

    op: VariableOr[ConditionTaskOp]
    """
    * `EQUAL_TO`, `NOT_EQUAL` operators perform string comparison of their operands. This means that `“12.0” == “12”` will evaluate to `false`.
    * `GREATER_THAN`, `GREATER_THAN_OR_EQUAL`, `LESS_THAN`, `LESS_THAN_OR_EQUAL` operators perform numeric comparison of their operands. `“12.0” >= “12”` will evaluate to `true`, `“10.0” >= “12”` will evaluate to `false`.
    
    The boolean comparison to task values can be implemented with operators `EQUAL_TO`, `NOT_EQUAL`. If a task value was set to a boolean value, it will be serialized to `“true”` or `“false”` for the comparison.
    """

    right: VariableOr[str]
    """
    The right operand of the condition task. Can be either a string value or a job state or parameter reference.
    """

    @classmethod
    def from_dict(cls, value: "ConditionTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ConditionTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class ConditionTaskDict(TypedDict, total=False):
    """"""

    left: VariableOr[str]
    """
    The left operand of the condition task. Can be either a string value or a job state or parameter reference.
    """

    op: VariableOr[ConditionTaskOpParam]
    """
    * `EQUAL_TO`, `NOT_EQUAL` operators perform string comparison of their operands. This means that `“12.0” == “12”` will evaluate to `false`.
    * `GREATER_THAN`, `GREATER_THAN_OR_EQUAL`, `LESS_THAN`, `LESS_THAN_OR_EQUAL` operators perform numeric comparison of their operands. `“12.0” >= “12”` will evaluate to `true`, `“10.0” >= “12”` will evaluate to `false`.
    
    The boolean comparison to task values can be implemented with operators `EQUAL_TO`, `NOT_EQUAL`. If a task value was set to a boolean value, it will be serialized to `“true”` or `“false”` for the comparison.
    """

    right: VariableOr[str]
    """
    The right operand of the condition task. Can be either a string value or a job state or parameter reference.
    """


ConditionTaskParam = ConditionTaskDict | ConditionTask
