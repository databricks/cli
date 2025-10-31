from enum import Enum
from typing import Literal


class AppDeploymentState(Enum):
    SUCCEEDED = "SUCCEEDED"
    FAILED = "FAILED"
    IN_PROGRESS = "IN_PROGRESS"
    CANCELLED = "CANCELLED"


AppDeploymentStateParam = (
    Literal["SUCCEEDED", "FAILED", "IN_PROGRESS", "CANCELLED"] | AppDeploymentState
)
