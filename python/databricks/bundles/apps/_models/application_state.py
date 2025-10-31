from enum import Enum
from typing import Literal


class ApplicationState(Enum):
    DEPLOYING = "DEPLOYING"
    RUNNING = "RUNNING"
    CRASHED = "CRASHED"
    UNAVAILABLE = "UNAVAILABLE"


ApplicationStateParam = (
    Literal["DEPLOYING", "RUNNING", "CRASHED", "UNAVAILABLE"] | ApplicationState
)
