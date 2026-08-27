from enum import Enum
from typing import Literal


class SchedulePauseStatus(Enum):
    UNPAUSED = "UNPAUSED"
    PAUSED = "PAUSED"


SchedulePauseStatusParam = Literal["UNPAUSED", "PAUSED"] | SchedulePauseStatus
