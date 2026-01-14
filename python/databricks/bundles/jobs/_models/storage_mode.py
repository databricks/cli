from enum import Enum
from typing import Literal


class StorageMode(Enum):
    """
    Valid values are: `DIRECT_QUERY`, `IMPORT`, and `DUAL`.
    """

    DIRECT_QUERY = "DIRECT_QUERY"
    IMPORT = "IMPORT"
    DUAL = "DUAL"


StorageModeParam = Literal["DIRECT_QUERY", "IMPORT", "DUAL"] | StorageMode
