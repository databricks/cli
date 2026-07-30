from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TaskDependency:
    """"""

    task_key: VariableOr[str]
    """
    The name of the task this task depends on.
    """

    outcome: VariableOrOptional[str] = None
    """
    Can only be specified on condition task dependencies. The outcome of the dependent task that must be met for this task to run.
    """

    @classmethod
    def from_dict(cls, value: "TaskDependencyDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TaskDependencyDict":
        return _transform_to_json_value(self)  # type:ignore


class TaskDependencyDict(TypedDict, total=False):
    """"""

    task_key: VariableOr[str]
    """
    The name of the task this task depends on.
    """

    outcome: VariableOrOptional[str]
    """
    Can only be specified on condition task dependencies. The outcome of the dependent task that must be met for this task to run.
    """


TaskDependencyParam = TaskDependencyDict | TaskDependency
