from enum import Enum
from typing import Literal


class Kind(Enum):
    CLASSIC_PREVIEW = "CLASSIC_PREVIEW"


KindParam = Literal["CLASSIC_PREVIEW"] | Kind
