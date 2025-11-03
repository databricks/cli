from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.source import Source, SourceParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SparkPythonTask:
    """"""

    python_file: VariableOr[str]
    """
    The Python file to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    parameters: VariableOrList[str] = field(default_factory=list)
    """
    Command line parameters passed to the Python file.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    source: VariableOrOptional[Source] = None
    """
    Optional location type of the Python file. When set to `WORKSPACE` or not specified, the file will be retrieved from the local
    Databricks workspace or cloud location (if the `python_file` has a URI format). When set to `GIT`,
    the Python file will be retrieved from a Git repository defined in `git_source`.
    
    * `WORKSPACE`: The Python file is located in a Databricks workspace or at a cloud filesystem URI.
    * `GIT`: The Python file is located in a remote Git repository.
    """

    @classmethod
    def from_dict(cls, value: "SparkPythonTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SparkPythonTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class SparkPythonTaskDict(TypedDict, total=False):
    """"""

    python_file: VariableOr[str]
    """
    The Python file to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    parameters: VariableOrList[str]
    """
    Command line parameters passed to the Python file.
    
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    """

    source: VariableOrOptional[SourceParam]
    """
    Optional location type of the Python file. When set to `WORKSPACE` or not specified, the file will be retrieved from the local
    Databricks workspace or cloud location (if the `python_file` has a URI format). When set to `GIT`,
    the Python file will be retrieved from a Git repository defined in `git_source`.
    
    * `WORKSPACE`: The Python file is located in a Databricks workspace or at a cloud filesystem URI.
    * `GIT`: The Python file is located in a remote Git repository.
    """


SparkPythonTaskParam = SparkPythonTaskDict | SparkPythonTask
