from enum import Enum
from typing import Literal


class DatabaseInstancePermissionLevel(Enum):
    CAN_CREATE = "CAN_CREATE"
    CAN_USE = "CAN_USE"
    CAN_MANAGE = "CAN_MANAGE"


DatabaseInstancePermissionLevelParam = (
    Literal["CAN_CREATE", "CAN_USE", "CAN_MANAGE"] | DatabaseInstancePermissionLevel
)
