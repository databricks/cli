from enum import Enum
from typing import Literal


class SchemaGrantPrivilege(Enum):
    ALL_PRIVILEGES = "ALL_PRIVILEGES"
    APPLY_TAG = "APPLY_TAG"
    CREATE_FUNCTION = "CREATE_FUNCTION"
    CREATE_TABLE = "CREATE_TABLE"
    CREATE_VOLUME = "CREATE_VOLUME"
    MANAGE = "MANAGE"
    USE_SCHEMA = "USE_SCHEMA"
    EXECUTE = "EXECUTE"
    MODIFY = "MODIFY"
    REFRESH = "REFRESH"
    SELECT = "SELECT"
    READ_VOLUME = "READ_VOLUME"
    WRITE_VOLUME = "WRITE_VOLUME"


SchemaGrantPrivilegeParam = (
    Literal[
        "ALL_PRIVILEGES",
        "APPLY_TAG",
        "CREATE_FUNCTION",
        "CREATE_TABLE",
        "CREATE_VOLUME",
        "MANAGE",
        "USE_SCHEMA",
        "EXECUTE",
        "MODIFY",
        "REFRESH",
        "SELECT",
        "READ_VOLUME",
        "WRITE_VOLUME",
    ]
    | SchemaGrantPrivilege
)
