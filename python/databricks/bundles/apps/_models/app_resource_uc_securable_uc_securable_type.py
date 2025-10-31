from enum import Enum
from typing import Literal


class AppResourceUcSecurableUcSecurableType(Enum):
    VOLUME = "VOLUME"


AppResourceUcSecurableUcSecurableTypeParam = (
    Literal["VOLUME"] | AppResourceUcSecurableUcSecurableType
)
