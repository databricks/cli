from enum import Enum
from typing import Literal


class SharepointOptionsSharepointEntityType(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    FILE = "FILE"
    FILE_METADATA = "FILE_METADATA"
    PERMISSION = "PERMISSION"
    LIST = "LIST"


SharepointOptionsSharepointEntityTypeParam = (
    Literal["FILE", "FILE_METADATA", "PERMISSION", "LIST"]
    | SharepointOptionsSharepointEntityType
)
