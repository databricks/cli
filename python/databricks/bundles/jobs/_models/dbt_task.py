from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.source import Source, SourceParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DbtTask:
    """"""

    catalog: VariableOrOptional[str] = None
    """
    Optional name of the catalog to use. The value is the top level in the 3-level namespace of Unity Catalog (catalog / schema / relation). The catalog value can only be specified if a warehouse_id is specified. Requires dbt-databricks >= 1.1.1.
    """

    commands: VariableOrList[str] = field(default_factory=list)
    """
    A list of dbt commands to execute. All commands must start with `dbt`. This parameter must not be empty. A maximum of up to 10 commands can be provided.
    """

    profiles_directory: VariableOrOptional[str] = None
    """
    Optional (relative) path to the profiles directory. Can only be specified if no warehouse_id is specified. If no warehouse_id is specified and this folder is unset, the root directory is used.
    """

    project_directory: VariableOrOptional[str] = None
    """
    Path to the project directory. Optional for Git sourced tasks, in which
    case if no value is provided, the root of the Git repository is used.
    """

    schema: VariableOrOptional[str] = None
    """
    Optional schema to write to. This parameter is only used when a warehouse_id is also provided. If not provided, the `default` schema is used.
    """

    source: VariableOrOptional[Source] = None
    """
    Optional location type of the project directory. When set to `WORKSPACE`, the project will be retrieved
    from the local Databricks workspace. When set to `GIT`, the project will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    
    * `WORKSPACE`: Project is located in Databricks workspace.
    * `GIT`: Project is located in cloud Git provider.
    """

    warehouse_id: VariableOrOptional[str] = None
    """
    ID of the SQL warehouse to connect to. If provided, we automatically generate and provide the profile and connection details to dbt. It can be overridden on a per-command basis by using the `--profiles-dir` command line argument.
    """

    @classmethod
    def from_dict(cls, value: "DbtTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DbtTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class DbtTaskDict(TypedDict, total=False):
    """"""

    catalog: VariableOrOptional[str]
    """
    Optional name of the catalog to use. The value is the top level in the 3-level namespace of Unity Catalog (catalog / schema / relation). The catalog value can only be specified if a warehouse_id is specified. Requires dbt-databricks >= 1.1.1.
    """

    commands: VariableOrList[str]
    """
    A list of dbt commands to execute. All commands must start with `dbt`. This parameter must not be empty. A maximum of up to 10 commands can be provided.
    """

    profiles_directory: VariableOrOptional[str]
    """
    Optional (relative) path to the profiles directory. Can only be specified if no warehouse_id is specified. If no warehouse_id is specified and this folder is unset, the root directory is used.
    """

    project_directory: VariableOrOptional[str]
    """
    Path to the project directory. Optional for Git sourced tasks, in which
    case if no value is provided, the root of the Git repository is used.
    """

    schema: VariableOrOptional[str]
    """
    Optional schema to write to. This parameter is only used when a warehouse_id is also provided. If not provided, the `default` schema is used.
    """

    source: VariableOrOptional[SourceParam]
    """
    Optional location type of the project directory. When set to `WORKSPACE`, the project will be retrieved
    from the local Databricks workspace. When set to `GIT`, the project will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    
    * `WORKSPACE`: Project is located in Databricks workspace.
    * `GIT`: Project is located in cloud Git provider.
    """

    warehouse_id: VariableOrOptional[str]
    """
    ID of the SQL warehouse to connect to. If provided, we automatically generate and provide the profile and connection details to dbt. It can be overridden on a per-command basis by using the `--profiles-dir` command line argument.
    """


DbtTaskParam = DbtTaskDict | DbtTask
