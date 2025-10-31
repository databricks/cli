from enum import Enum
from typing import Literal


class AppResourceJobJobPermission(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    IS_OWNER = "IS_OWNER"
    CAN_MANAGE_RUN = "CAN_MANAGE_RUN"
    CAN_VIEW = "CAN_VIEW"


AppResourceJobJobPermissionParam = (
    Literal["CAN_MANAGE", "IS_OWNER", "CAN_MANAGE_RUN", "CAN_VIEW"]
    | AppResourceJobJobPermission
)
