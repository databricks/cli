from enum import Enum
from typing import Literal


class Aggregation(Enum):
    SUM = "SUM"
    COUNT = "COUNT"
    COUNT_DISTINCT = "COUNT_DISTINCT"
    AVG = "AVG"
    MEDIAN = "MEDIAN"
    MIN = "MIN"
    MAX = "MAX"
    STDDEV = "STDDEV"


AggregationParam = (
    Literal["SUM", "COUNT", "COUNT_DISTINCT", "AVG", "MEDIAN", "MIN", "MAX", "STDDEV"]
    | Aggregation
)
