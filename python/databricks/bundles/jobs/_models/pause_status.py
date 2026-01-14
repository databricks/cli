from enum import Enum
from typing import Literal


class PauseStatus(Enum):
    """
    Valid values are: `UNPAUSED` and `PAUSED`.
    """

    UNPAUSED = "UNPAUSED"
    PAUSED = "PAUSED"


PauseStatusParam = Literal["UNPAUSED", "PAUSED"] | PauseStatus
