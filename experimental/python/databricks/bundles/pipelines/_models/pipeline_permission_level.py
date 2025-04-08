from enum import Enum
from typing import Literal


class PipelinePermissionLevel(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    IS_OWNER = "IS_OWNER"
    CAN_RUN = "CAN_RUN"
    CAN_VIEW = "CAN_VIEW"


PipelinePermissionLevelParam = (
    Literal["CAN_MANAGE", "IS_OWNER", "CAN_RUN", "CAN_VIEW"] | PipelinePermissionLevel
)
