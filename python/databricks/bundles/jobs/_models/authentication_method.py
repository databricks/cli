from enum import Enum
from typing import Literal


class AuthenticationMethod(Enum):
    OAUTH = "OAUTH"
    PAT = "PAT"


AuthenticationMethodParam = Literal["OAUTH", "PAT"] | AuthenticationMethod
