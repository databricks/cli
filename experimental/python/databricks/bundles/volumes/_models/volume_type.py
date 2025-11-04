from enum import Enum
from typing import Literal


class VolumeType(Enum):
    MANAGED = "MANAGED"
    EXTERNAL = "EXTERNAL"


VolumeTypeParam = Literal["MANAGED", "EXTERNAL"] | VolumeType
