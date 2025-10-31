from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.source import Source, SourceParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class NotebookTask:
    """"""

    notebook_path: VariableOr[str]
    """
    The path of the notebook to be run in the Databricks workspace or remote repository.
    For notebooks stored in the Databricks workspace, the path must be absolute and begin with a slash.
    For notebooks stored in a remote repository, the path must be relative. This field is required.
    """

    base_parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    Base parameters to be used for each run of this job. If the run is initiated by a call to :method:jobs/run
    Now with parameters specified, the two parameters maps are merged. If the same key is specified in
    `base_parameters` and in `run-now`, the value from `run-now` is used.
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    
    If the notebook takes a parameter that is not specified in the job’s `base_parameters` or the `run-now` override parameters,
    the default value from the notebook is used.
    
    Retrieve these parameters in a notebook using [dbutils.widgets.get](https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-widgets).
    
    The JSON representation of this field cannot exceed 1MB.
    """

    source: VariableOrOptional[Source] = None
    """
    Optional location type of the notebook. When set to `WORKSPACE`, the notebook will be retrieved from the local Databricks workspace. When set to `GIT`, the notebook will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Notebook is located in Databricks workspace.
    * `GIT`: Notebook is located in cloud Git provider.
    """

    warehouse_id: VariableOrOptional[str] = None
    """
    Optional `warehouse_id` to run the notebook on a SQL warehouse. Classic SQL warehouses are NOT supported, please use serverless or pro SQL warehouses.
    
    Note that SQL warehouses only support SQL cells; if the notebook contains non-SQL cells, the run will fail.
    """

    @classmethod
    def from_dict(cls, value: "NotebookTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "NotebookTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class NotebookTaskDict(TypedDict, total=False):
    """"""

    notebook_path: VariableOr[str]
    """
    The path of the notebook to be run in the Databricks workspace or remote repository.
    For notebooks stored in the Databricks workspace, the path must be absolute and begin with a slash.
    For notebooks stored in a remote repository, the path must be relative. This field is required.
    """

    base_parameters: VariableOrDict[str]
    """
    Base parameters to be used for each run of this job. If the run is initiated by a call to :method:jobs/run
    Now with parameters specified, the two parameters maps are merged. If the same key is specified in
    `base_parameters` and in `run-now`, the value from `run-now` is used.
    Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
    
    If the notebook takes a parameter that is not specified in the job’s `base_parameters` or the `run-now` override parameters,
    the default value from the notebook is used.
    
    Retrieve these parameters in a notebook using [dbutils.widgets.get](https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-widgets).
    
    The JSON representation of this field cannot exceed 1MB.
    """

    source: VariableOrOptional[SourceParam]
    """
    Optional location type of the notebook. When set to `WORKSPACE`, the notebook will be retrieved from the local Databricks workspace. When set to `GIT`, the notebook will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Notebook is located in Databricks workspace.
    * `GIT`: Notebook is located in cloud Git provider.
    """

    warehouse_id: VariableOrOptional[str]
    """
    Optional `warehouse_id` to run the notebook on a SQL warehouse. Classic SQL warehouses are NOT supported, please use serverless or pro SQL warehouses.
    
    Note that SQL warehouses only support SQL cells; if the notebook contains non-SQL cells, the run will fail.
    """


NotebookTaskParam = NotebookTaskDict | NotebookTask
