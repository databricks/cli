from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.source import Source, SourceParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SqlTaskFile:
    """"""

    path: VariableOr[str]
    """
    Path of the SQL file. Must be relative if the source is a remote Git repository and absolute for workspace paths.
    """

    source: VariableOrOptional[Source] = None
    """
    Optional location type of the SQL file. When set to `WORKSPACE`, the SQL file will be retrieved
    from the local Databricks workspace. When set to `GIT`, the SQL file will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    
    * `WORKSPACE`: SQL file is located in Databricks workspace.
    * `GIT`: SQL file is located in cloud Git provider.
    """

    @classmethod
    def from_dict(cls, value: "SqlTaskFileDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SqlTaskFileDict":
        return _transform_to_json_value(self)  # type:ignore


class SqlTaskFileDict(TypedDict, total=False):
    """"""

    path: VariableOr[str]
    """
    Path of the SQL file. Must be relative if the source is a remote Git repository and absolute for workspace paths.
    """

    source: VariableOrOptional[SourceParam]
    """
    Optional location type of the SQL file. When set to `WORKSPACE`, the SQL file will be retrieved
    from the local Databricks workspace. When set to `GIT`, the SQL file will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    
    * `WORKSPACE`: SQL file is located in Databricks workspace.
    * `GIT`: SQL file is located in cloud Git provider.
    """


SqlTaskFileParam = SqlTaskFileDict | SqlTaskFile
