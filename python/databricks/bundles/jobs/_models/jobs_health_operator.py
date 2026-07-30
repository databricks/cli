from enum import Enum
from typing import Literal


class JobsHealthOperator(Enum):
    """
    Specifies the operator used to compare the health metric value with the specified threshold.
    """

    GREATER_THAN = "GREATER_THAN"


JobsHealthOperatorParam = Literal["GREATER_THAN"] | JobsHealthOperator
