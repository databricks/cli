from enum import Enum
from typing import Literal


class AiGatewayRateLimitRenewalPeriod(Enum):
    MINUTE = "minute"


AiGatewayRateLimitRenewalPeriodParam = (
    Literal["minute"] | AiGatewayRateLimitRenewalPeriod
)
