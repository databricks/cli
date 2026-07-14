from enum import Enum
from typing import Literal


class GoogleDriveOptionsGoogleDriveEntityType(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    FILE = "FILE"
    FILE_METADATA = "FILE_METADATA"
    PERMISSION = "PERMISSION"


GoogleDriveOptionsGoogleDriveEntityTypeParam = (
    Literal["FILE", "FILE_METADATA", "PERMISSION"]
    | GoogleDriveOptionsGoogleDriveEntityType
)
