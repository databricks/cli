from enum import Enum
from typing import Literal


class AppDeploymentMode(Enum):
    SNAPSHOT = "SNAPSHOT"
    AUTO_SYNC = "AUTO_SYNC"


AppDeploymentModeParam = Literal["SNAPSHOT", "AUTO_SYNC"] | AppDeploymentMode
