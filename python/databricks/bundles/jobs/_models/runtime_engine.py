from enum import Enum
from typing import Literal


class RuntimeEngine(Enum):
    NULL = "NULL"
    STANDARD = "STANDARD"
    PHOTON = "PHOTON"


RuntimeEngineParam = Literal["NULL", "STANDARD", "PHOTON"] | RuntimeEngine
