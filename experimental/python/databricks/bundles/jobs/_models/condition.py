from enum import Enum
from typing import Literal


class Condition(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    ANY_UPDATED = "ANY_UPDATED"
    ALL_UPDATED = "ALL_UPDATED"


ConditionParam = Literal["ANY_UPDATED", "ALL_UPDATED"] | Condition
