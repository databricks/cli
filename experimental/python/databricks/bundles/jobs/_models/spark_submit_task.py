from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SparkSubmitTask:
    """"""

    parameters: VariableOrList[str] = field(default_factory=list)
    """
    Command-line parameters passed to spark submit.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    @classmethod
    def from_dict(cls, value: "SparkSubmitTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SparkSubmitTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class SparkSubmitTaskDict(TypedDict, total=False):
    """"""

    parameters: VariableOrList[str]
    """
    Command-line parameters passed to spark submit.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """


SparkSubmitTaskParam = SparkSubmitTaskDict | SparkSubmitTask
