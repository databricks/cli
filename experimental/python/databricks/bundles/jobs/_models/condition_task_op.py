from enum import Enum
from typing import Literal


class ConditionTaskOp(Enum):
    """
    * `EQUAL_TO`, `NOT_EQUAL` operators perform string comparison of their operands. This means that `“12.0” == “12”` will evaluate to `false`.
    * `GREATER_THAN`, `GREATER_THAN_OR_EQUAL`, `LESS_THAN`, `LESS_THAN_OR_EQUAL` operators perform numeric comparison of their operands. `“12.0” >= “12”` will evaluate to `true`, `“10.0” >= “12”` will evaluate to `false`.

    The boolean comparison to task values can be implemented with operators `EQUAL_TO`, `NOT_EQUAL`. If a task value was set to a boolean value, it will be serialized to `“true”` or `“false”` for the comparison.
    """

    EQUAL_TO = "EQUAL_TO"
    GREATER_THAN = "GREATER_THAN"
    GREATER_THAN_OR_EQUAL = "GREATER_THAN_OR_EQUAL"
    LESS_THAN = "LESS_THAN"
    LESS_THAN_OR_EQUAL = "LESS_THAN_OR_EQUAL"
    NOT_EQUAL = "NOT_EQUAL"


ConditionTaskOpParam = (
    Literal[
        "EQUAL_TO",
        "GREATER_THAN",
        "GREATER_THAN_OR_EQUAL",
        "LESS_THAN",
        "LESS_THAN_OR_EQUAL",
        "NOT_EQUAL",
    ]
    | ConditionTaskOp
)
