from enum import Enum
from typing import Literal


class DayOfWeek(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Days of week in which the window is allowed to happen.
    If not specified all days of the week will be used.
    """

    MONDAY = "MONDAY"
    TUESDAY = "TUESDAY"
    WEDNESDAY = "WEDNESDAY"
    THURSDAY = "THURSDAY"
    FRIDAY = "FRIDAY"
    SATURDAY = "SATURDAY"
    SUNDAY = "SUNDAY"


DayOfWeekParam = (
    Literal[
        "MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"
    ]
    | DayOfWeek
)
