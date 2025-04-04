from enum import Enum
from typing import Literal


class DeploymentKind(Enum):
    """
    The deployment method that manages the pipeline:
    - BUNDLE: The pipeline is managed by a Databricks Asset Bundle.

    """

    BUNDLE = "BUNDLE"


DeploymentKindParam = Literal["BUNDLE"] | DeploymentKind
