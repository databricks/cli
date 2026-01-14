from enum import Enum
from typing import Literal


class VolumeType(Enum):
    """
    Valid values are: `MANAGED` and `EXTERNAL`.
    """

    MANAGED = "MANAGED"
    EXTERNAL = "EXTERNAL"


VolumeTypeParam = Literal["MANAGED", "EXTERNAL"] | VolumeType
