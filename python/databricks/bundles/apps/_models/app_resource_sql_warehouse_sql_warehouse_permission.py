from enum import Enum
from typing import Literal


class AppResourceSqlWarehouseSqlWarehousePermission(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_USE = "CAN_USE"
    IS_OWNER = "IS_OWNER"


AppResourceSqlWarehouseSqlWarehousePermissionParam = (
    Literal["CAN_MANAGE", "CAN_USE", "IS_OWNER"]
    | AppResourceSqlWarehouseSqlWarehousePermission
)
