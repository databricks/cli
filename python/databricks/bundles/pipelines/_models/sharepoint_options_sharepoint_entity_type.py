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
    FILE_PERMISSION = "FILE_PERMISSION"
    GROUP_MEMBERSHIP = "GROUP_MEMBERSHIP"


SharepointOptionsSharepointEntityTypeParam = (
    Literal[
        "FILE",
        "FILE_METADATA",
        "PERMISSION",
        "LIST",
        "FILE_PERMISSION",
        "GROUP_MEMBERSHIP",
    ]
    | SharepointOptionsSharepointEntityType
)
