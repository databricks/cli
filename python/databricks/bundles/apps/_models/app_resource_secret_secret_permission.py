from enum import Enum
from typing import Literal


class AppResourceSecretSecretPermission(Enum):
    """
    Permission to grant on the secret scope. Supported permissions are: "READ", "WRITE", "MANAGE".
    """

    READ = "READ"
    WRITE = "WRITE"
    MANAGE = "MANAGE"


AppResourceSecretSecretPermissionParam = (
    Literal["READ", "WRITE", "MANAGE"] | AppResourceSecretSecretPermission
)
