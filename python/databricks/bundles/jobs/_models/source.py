from enum import Enum
from typing import Literal


class Source(Enum):
    """
    Optional location type of the SQL file. When set to `WORKSPACE`, the SQL file will be retrieved\
    from the local Databricks workspace. When set to `GIT`, the SQL file will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    
    * `WORKSPACE`: SQL file is located in Databricks workspace.
    * `GIT`: SQL file is located in cloud Git provider.
    """

    WORKSPACE = "WORKSPACE"
    GIT = "GIT"


SourceParam = Literal["WORKSPACE", "GIT"] | Source
