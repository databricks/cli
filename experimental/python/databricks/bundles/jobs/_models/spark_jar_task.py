from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SparkJarTask:
    """"""

    main_class_name: VariableOr[str]
    """
    The full name of the class containing the main method to be executed. This class must be contained in a JAR provided as a library.
    
    The code must use `SparkContext.getOrCreate` to obtain a Spark context; otherwise, runs of the job fail.
    """

    parameters: VariableOrList[str] = field(default_factory=list)
    """
    Parameters passed to the main method.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    @classmethod
    def from_dict(cls, value: "SparkJarTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SparkJarTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class SparkJarTaskDict(TypedDict, total=False):
    """"""

    main_class_name: VariableOr[str]
    """
    The full name of the class containing the main method to be executed. This class must be contained in a JAR provided as a library.
    
    The code must use `SparkContext.getOrCreate` to obtain a Spark context; otherwise, runs of the job fail.
    """

    parameters: VariableOrList[str]
    """
    Parameters passed to the main method.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """


SparkJarTaskParam = SparkJarTaskDict | SparkJarTask
