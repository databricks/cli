from enum import Enum
from typing import Literal


class PauseStatus(Enum):
    UNPAUSED = "UNPAUSED"
    PAUSED = "PAUSED"


PauseStatusParam = Literal["UNPAUSED", "PAUSED"] | PauseStatus
