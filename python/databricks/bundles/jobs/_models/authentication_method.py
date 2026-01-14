from enum import Enum
from typing import Literal


class AuthenticationMethod(Enum):
    """
    Valid values are: `OAUTH` and `PAT`.
    """

    OAUTH = "OAUTH"
    PAT = "PAT"


AuthenticationMethodParam = Literal["OAUTH", "PAT"] | AuthenticationMethod
