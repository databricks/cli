from enum import Enum
from typing import Literal


class ModelServingEndpointPermissionLevel(Enum):
    CAN_MANAGE = "CAN_MANAGE"
    CAN_QUERY = "CAN_QUERY"
    CAN_VIEW = "CAN_VIEW"


ModelServingEndpointPermissionLevelParam = (
    Literal["CAN_MANAGE", "CAN_QUERY", "CAN_VIEW"] | ModelServingEndpointPermissionLevel
)
