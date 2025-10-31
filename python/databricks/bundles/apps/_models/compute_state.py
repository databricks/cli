from enum import Enum
from typing import Literal


class ComputeState(Enum):
    ERROR = "ERROR"
    DELETING = "DELETING"
    STARTING = "STARTING"
    STOPPING = "STOPPING"
    UPDATING = "UPDATING"
    STOPPED = "STOPPED"
    ACTIVE = "ACTIVE"


ComputeStateParam = (
    Literal[
        "ERROR", "DELETING", "STARTING", "STOPPING", "UPDATING", "STOPPED", "ACTIVE"
    ]
    | ComputeState
)
