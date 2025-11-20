from enum import Enum
from typing import Literal


class TaskRetryMode(Enum):
    """
    task retry mode of the continuous job
    * NEVER: The failed task will not be retried.
    * ON_FAILURE: Retry a failed task if at least one other task in the job is still running its first attempt.
    When this condition is no longer met or the retry limit is reached, the job run is cancelled and a new run is started.
    """

    NEVER = "NEVER"
    ON_FAILURE = "ON_FAILURE"


TaskRetryModeParam = Literal["NEVER", "ON_FAILURE"] | TaskRetryMode
