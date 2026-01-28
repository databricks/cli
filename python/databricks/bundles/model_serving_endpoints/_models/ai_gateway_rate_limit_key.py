from enum import Enum
from typing import Literal


class AiGatewayRateLimitKey(Enum):
    USER = "user"
    ENDPOINT = "endpoint"
    USER_GROUP = "user_group"
    SERVICE_PRINCIPAL = "service_principal"


AiGatewayRateLimitKeyParam = (
    Literal["user", "endpoint", "user_group", "service_principal"]
    | AiGatewayRateLimitKey
)
