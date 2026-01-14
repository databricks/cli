from enum import Enum
from typing import Literal


class Condition(Enum):
    """
    Valid values are: `ANY_UPDATED` and `ALL_UPDATED`.
    """

    ANY_UPDATED = "ANY_UPDATED"
    ALL_UPDATED = "ALL_UPDATED"


ConditionParam = Literal["ANY_UPDATED", "ALL_UPDATED"] | Condition
