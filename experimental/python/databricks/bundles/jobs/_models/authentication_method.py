from enum import Enum
from typing import Literal


class AuthenticationMethod(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    OAUTH = "OAUTH"
    PAT = "PAT"


AuthenticationMethodParam = Literal["OAUTH", "PAT"] | AuthenticationMethod
