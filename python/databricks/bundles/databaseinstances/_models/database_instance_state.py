from enum import Enum
from typing import Literal


class DatabaseInstanceState(Enum):
    STARTING = "STARTING"
    AVAILABLE = "AVAILABLE"
    DELETING = "DELETING"
    STOPPED = "STOPPED"
    UPDATING = "UPDATING"
    FAILING_OVER = "FAILING_OVER"


DatabaseInstanceStateParam = (
    Literal["STARTING", "AVAILABLE", "DELETING", "STOPPED", "UPDATING", "FAILING_OVER"]
    | DatabaseInstanceState
)
