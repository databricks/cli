from enum import Enum
from typing import Literal


class DayOfWeek(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour).
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
