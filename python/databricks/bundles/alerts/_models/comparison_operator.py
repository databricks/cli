from enum import Enum
from typing import Literal


class ComparisonOperator(Enum):
    LESS_THAN = "LESS_THAN"
    GREATER_THAN = "GREATER_THAN"
    EQUAL = "EQUAL"
    NOT_EQUAL = "NOT_EQUAL"
    GREATER_THAN_OR_EQUAL = "GREATER_THAN_OR_EQUAL"
    LESS_THAN_OR_EQUAL = "LESS_THAN_OR_EQUAL"
    IS_NULL = "IS_NULL"
    IS_NOT_NULL = "IS_NOT_NULL"


ComparisonOperatorParam = (
    Literal[
        "LESS_THAN",
        "GREATER_THAN",
        "EQUAL",
        "NOT_EQUAL",
        "GREATER_THAN_OR_EQUAL",
        "LESS_THAN_OR_EQUAL",
        "IS_NULL",
        "IS_NOT_NULL",
    ]
    | ComparisonOperator
)
