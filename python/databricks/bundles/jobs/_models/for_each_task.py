from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self

    from databricks.bundles.jobs._models.task import Task, TaskParam


@dataclass(kw_only=True)
class ForEachTask:
    """"""

    inputs: VariableOr[str]
    """
    Array for task to iterate on. This can be a JSON string or a reference to
    an array parameter.
    """

    task: VariableOr["Task"]
    """
    Configuration for the task that will be run for each element in the array
    """

    concurrency: VariableOrOptional[int] = None
    """
    An optional maximum allowed number of concurrent runs of the task.
    Set this value if you want to be able to execute multiple runs of the task concurrently.
    """

    @classmethod
    def from_dict(cls, value: "ForEachTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ForEachTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class ForEachTaskDict(TypedDict, total=False):
    """"""

    inputs: VariableOr[str]
    """
    Array for task to iterate on. This can be a JSON string or a reference to
    an array parameter.
    """

    task: VariableOr["TaskParam"]
    """
    Configuration for the task that will be run for each element in the array
    """

    concurrency: VariableOrOptional[int]
    """
    An optional maximum allowed number of concurrent runs of the task.
    Set this value if you want to be able to execute multiple runs of the task concurrently.
    """


ForEachTaskParam = ForEachTaskDict | ForEachTask
