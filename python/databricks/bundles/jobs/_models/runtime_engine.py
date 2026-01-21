from enum import Enum
from typing import Literal


class RuntimeEngine(Enum):
    """
    Valid values are: `NULL`, `STANDARD`, and `PHOTON`.
    """

    NULL = "NULL"
    STANDARD = "STANDARD"
    PHOTON = "PHOTON"


RuntimeEngineParam = Literal["NULL", "STANDARD", "PHOTON"] | RuntimeEngine
