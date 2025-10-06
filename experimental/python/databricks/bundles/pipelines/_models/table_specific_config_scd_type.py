from enum import Enum
from typing import Literal


class TableSpecificConfigScdType(Enum):
    """
    :meta private: [EXPERIMENTAL]

    The SCD type to use to ingest the table.
    """

    SCD_TYPE_1 = "SCD_TYPE_1"
    SCD_TYPE_2 = "SCD_TYPE_2"
    APPEND_ONLY = "APPEND_ONLY"


TableSpecificConfigScdTypeParam = (
    Literal["SCD_TYPE_1", "SCD_TYPE_2", "APPEND_ONLY"] | TableSpecificConfigScdType
)
