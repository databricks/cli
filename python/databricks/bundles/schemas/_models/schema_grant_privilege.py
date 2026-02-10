from enum import Enum
from typing import Literal


class SchemaGrantPrivilege(Enum):
    ALL_PRIVILEGES = "ALL_PRIVILEGES"
    APPLY_TAG = "APPLY_TAG"
    CREATE_FUNCTION = "CREATE_FUNCTION"
    CREATE_MATERIALIZED_VIEW = "CREATE_MATERIALIZED_VIEW"
    CREATE_MODEL = "CREATE_MODEL"
    CREATE_TABLE = "CREATE_TABLE"
    CREATE_VOLUME = "CREATE_VOLUME"
    EXECUTE = "EXECUTE"
    EXTERNAL_USE_SCHEMA = "EXTERNAL_USE_SCHEMA"
    MANAGE = "MANAGE"
    MODIFY = "MODIFY"
    READ_VOLUME = "READ_VOLUME"
    REFRESH = "REFRESH"
    SELECT = "SELECT"
    USE_SCHEMA = "USE_SCHEMA"
    WRITE_VOLUME = "WRITE_VOLUME"


SchemaGrantPrivilegeParam = (
    Literal[
        "ALL_PRIVILEGES",
        "APPLY_TAG",
        "CREATE_FUNCTION",
        "CREATE_MATERIALIZED_VIEW",
        "CREATE_MODEL",
        "CREATE_TABLE",
        "CREATE_VOLUME",
        "EXECUTE",
        "EXTERNAL_USE_SCHEMA",
        "MANAGE",
        "MODIFY",
        "READ_VOLUME",
        "REFRESH",
        "SELECT",
        "USE_SCHEMA",
        "WRITE_VOLUME",
    ]
    | SchemaGrantPrivilege
)
