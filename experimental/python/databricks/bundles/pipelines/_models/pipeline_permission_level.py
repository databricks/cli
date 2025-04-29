from enum import Enum
from typing import Literal


class PipelinePermissionLevel(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_RUN = "CAN_RUN"
    CAN_VIEW = "CAN_VIEW"
    IS_OWNER = "IS_OWNER"


PipelinePermissionLevelParam = (
    Literal["CAN_MANAGE", "CAN_RUN", "CAN_VIEW", "IS_OWNER"] | PipelinePermissionLevel
)
