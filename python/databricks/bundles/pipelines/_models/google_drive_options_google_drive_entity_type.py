from enum import Enum
from typing import Literal


class GoogleDriveOptionsGoogleDriveEntityType(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    FILE = "FILE"
    FILE_METADATA = "FILE_METADATA"
    PERMISSION = "PERMISSION"
    FILE_PERMISSION = "FILE_PERMISSION"
    GROUP_MEMBERSHIP = "GROUP_MEMBERSHIP"


GoogleDriveOptionsGoogleDriveEntityTypeParam = (
    Literal[
        "FILE", "FILE_METADATA", "PERMISSION", "FILE_PERMISSION", "GROUP_MEMBERSHIP"
    ]
    | GoogleDriveOptionsGoogleDriveEntityType
)
