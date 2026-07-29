from enum import Enum
from typing import Literal


class DependencyMode(Enum):
    """
    Controls dependency configuration for the cluster.

    * `DEPENDENCY_MODE_AUTO`: Databricks will choose the most appropriate dependency mode based on your compute configuration.
    * `DEPENDENCY_MODE_ENVIRONMENTS`: Enables a unified dependency management experience across classic and serverless, resulting in increased stability and performance. Supported only on DBR 19+ in Standard access mode.
    * `DEPENDENCY_MODE_CLUSTER_LIBRARIES`: Legacy mode: dependencies come from cluster libraries and init scripts.
    """

    DEPENDENCY_MODE_ENVIRONMENTS = "DEPENDENCY_MODE_ENVIRONMENTS"
    DEPENDENCY_MODE_CLUSTER_LIBRARIES = "DEPENDENCY_MODE_CLUSTER_LIBRARIES"
    DEPENDENCY_MODE_AUTO = "DEPENDENCY_MODE_AUTO"


DependencyModeParam = (
    Literal[
        "DEPENDENCY_MODE_ENVIRONMENTS",
        "DEPENDENCY_MODE_CLUSTER_LIBRARIES",
        "DEPENDENCY_MODE_AUTO",
    ]
    | DependencyMode
)
