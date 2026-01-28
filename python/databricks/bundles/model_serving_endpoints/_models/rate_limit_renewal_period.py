from enum import Enum
from typing import Literal


class RateLimitRenewalPeriod(Enum):
    """
    [DEPRECATED]
    """

    MINUTE = "minute"


RateLimitRenewalPeriodParam = Literal["minute"] | RateLimitRenewalPeriod
