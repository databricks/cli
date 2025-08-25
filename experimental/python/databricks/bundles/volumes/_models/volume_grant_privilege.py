from enum import Enum
from typing import Literal


class VolumeGrantPrivilege(Enum):
    ALL_PRIVILEGES = "ALL_PRIVILEGES"
    APPLY_TAG = "APPLY_TAG"
    MANAGE = "MANAGE"
    READ_VOLUME = "READ_VOLUME"
    WRITE_VOLUME = "WRITE_VOLUME"


VolumeGrantPrivilegeParam = (
    Literal["ALL_PRIVILEGES", "APPLY_TAG", "MANAGE", "READ_VOLUME", "WRITE_VOLUME"]
    | VolumeGrantPrivilege
)
