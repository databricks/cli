from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

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

    jar_uri: VariableOrOptional[str] = None
    """
    [DEPRECATED] Deprecated since 04/2016. For classic compute, provide a `jar` through the `libraries` field instead. For serverless compute, provide a `jar` though the `java_dependencies` field inside the `environments` list.
    
    See the examples of classic and serverless compute usage at the top of the page.
    """

    parameters: VariableOrList[str] = field(default_factory=list)
    """
    Parameters passed to the main method.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    run_as_repl: VariableOrOptional[bool] = None
    """
    [DEPRECATED] Deprecated. A value of `false` is no longer supported.
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

    jar_uri: VariableOrOptional[str]
    """
    [DEPRECATED] Deprecated since 04/2016. For classic compute, provide a `jar` through the `libraries` field instead. For serverless compute, provide a `jar` though the `java_dependencies` field inside the `environments` list.
    
    See the examples of classic and serverless compute usage at the top of the page.
    """

    parameters: VariableOrList[str]
    """
    Parameters passed to the main method.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    run_as_repl: VariableOrOptional[bool]
    """
    [DEPRECATED] Deprecated. A value of `false` is no longer supported.
    """


SparkJarTaskParam = SparkJarTaskDict | SparkJarTask
