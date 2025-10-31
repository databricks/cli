from enum import Enum
from typing import Literal


class JobPermissionLevel(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_MANAGE_RUN = "CAN_MANAGE_RUN"
    CAN_VIEW = "CAN_VIEW"
    IS_OWNER = "IS_OWNER"


JobPermissionLevelParam = (
    Literal["CAN_MANAGE", "CAN_MANAGE_RUN", "CAN_VIEW", "IS_OWNER"] | JobPermissionLevel
)
