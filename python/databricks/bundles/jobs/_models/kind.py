from enum import Enum
from typing import Literal


class Kind(Enum):
    """
    Valid values are: `CLASSIC_PREVIEW`.
    """

    CLASSIC_PREVIEW = "CLASSIC_PREVIEW"


KindParam = Literal["CLASSIC_PREVIEW"] | Kind
