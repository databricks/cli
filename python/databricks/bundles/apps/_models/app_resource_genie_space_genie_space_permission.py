from enum import Enum
from typing import Literal


class AppResourceGenieSpaceGenieSpacePermission(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_EDIT = "CAN_EDIT"
    CAN_RUN = "CAN_RUN"
    CAN_VIEW = "CAN_VIEW"


AppResourceGenieSpaceGenieSpacePermissionParam = (
    Literal["CAN_MANAGE", "CAN_EDIT", "CAN_RUN", "CAN_VIEW"]
    | AppResourceGenieSpaceGenieSpacePermission
)
