from enum import Enum
from typing import Literal


class AppResourceServingEndpointServingEndpointPermission(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_QUERY = "CAN_QUERY"
    CAN_VIEW = "CAN_VIEW"


AppResourceServingEndpointServingEndpointPermissionParam = (
    Literal["CAN_MANAGE", "CAN_QUERY", "CAN_VIEW"]
    | AppResourceServingEndpointServingEndpointPermission
)
