from enum import Enum
from typing import Literal


class RateLimitKey(Enum):
    """
    [DEPRECATED]
    """

    USER = "user"
    ENDPOINT = "endpoint"


RateLimitKeyParam = Literal["user", "endpoint"] | RateLimitKey
