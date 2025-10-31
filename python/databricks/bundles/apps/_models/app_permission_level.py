from enum import Enum
from typing import Literal


class AppPermissionLevel(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_USE = "CAN_USE"


AppPermissionLevelParam = Literal["CAN_MANAGE", "CAN_USE"] | AppPermissionLevel
