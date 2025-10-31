from enum import Enum
from typing import Literal


class AppResourceUcSecurableUcSecurablePermission(Enum):
    READ_VOLUME = "READ_VOLUME"
    WRITE_VOLUME = "WRITE_VOLUME"


AppResourceUcSecurableUcSecurablePermissionParam = (
    Literal["READ_VOLUME", "WRITE_VOLUME"] | AppResourceUcSecurableUcSecurablePermission
)
