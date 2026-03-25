from enum import Enum
from typing import Literal


class CatalogIsolationMode(Enum):
    OPEN = "OPEN"
    ISOLATED = "ISOLATED"


CatalogIsolationModeParam = Literal["OPEN", "ISOLATED"] | CatalogIsolationMode
