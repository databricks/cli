from enum import Enum
from typing import Literal


class ComputeSize(Enum):
    MEDIUM = "MEDIUM"
    LARGE = "LARGE"
    LIQUID = "LIQUID"


ComputeSizeParam = Literal["MEDIUM", "LARGE", "LIQUID"] | ComputeSize
