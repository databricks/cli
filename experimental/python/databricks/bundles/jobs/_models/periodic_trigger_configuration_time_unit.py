# Code generated from jsonschema.json. DO NOT EDIT.
from enum import Enum
from typing import Literal


class PeriodicTriggerConfigurationTimeUnit(Enum):
    HOURS = "HOURS"
    DAYS = "DAYS"
    WEEKS = "WEEKS"


PeriodicTriggerConfigurationTimeUnitParam = (
    Literal["HOURS", "DAYS", "WEEKS"] | PeriodicTriggerConfigurationTimeUnit
)
