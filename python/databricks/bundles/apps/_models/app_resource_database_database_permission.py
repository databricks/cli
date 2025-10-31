from enum import Enum
from typing import Literal


class AppResourceDatabaseDatabasePermission(Enum):
    CAN_CONNECT_AND_CREATE = "CAN_CONNECT_AND_CREATE"


AppResourceDatabaseDatabasePermissionParam = (
    Literal["CAN_CONNECT_AND_CREATE"] | AppResourceDatabaseDatabasePermission
)
