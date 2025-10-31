from enum import Enum
from typing import Literal


class RunIf(Enum):
    """
    An optional value indicating the condition that determines whether the task should be run once its dependencies have been completed. When omitted, defaults to `ALL_SUCCESS`.

    Possible values are:
    * `ALL_SUCCESS`: All dependencies have executed and succeeded
    * `AT_LEAST_ONE_SUCCESS`: At least one dependency has succeeded
    * `NONE_FAILED`: None of the dependencies have failed and at least one was executed
    * `ALL_DONE`: All dependencies have been completed
    * `AT_LEAST_ONE_FAILED`: At least one dependency failed
    * `ALL_FAILED`: ALl dependencies have failed
    """

    ALL_SUCCESS = "ALL_SUCCESS"
    ALL_DONE = "ALL_DONE"
    NONE_FAILED = "NONE_FAILED"
    AT_LEAST_ONE_SUCCESS = "AT_LEAST_ONE_SUCCESS"
    ALL_FAILED = "ALL_FAILED"
    AT_LEAST_ONE_FAILED = "AT_LEAST_ONE_FAILED"


RunIfParam = (
    Literal[
        "ALL_SUCCESS",
        "ALL_DONE",
        "NONE_FAILED",
        "AT_LEAST_ONE_SUCCESS",
        "ALL_FAILED",
        "AT_LEAST_ONE_FAILED",
    ]
    | RunIf
)
