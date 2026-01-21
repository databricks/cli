from enum import Enum
from typing import Literal


class PeriodicTriggerConfigurationTimeUnit(Enum):
    """
    Valid values are: `HOURS`, `DAYS`, and `WEEKS`.
    """

    HOURS = "HOURS"
    DAYS = "DAYS"
    WEEKS = "WEEKS"


PeriodicTriggerConfigurationTimeUnitParam = (
    Literal["HOURS", "DAYS", "WEEKS"] | PeriodicTriggerConfigurationTimeUnit
)
