from enum import Enum
from typing import Literal


class DataSecurityMode(Enum):
    """
    Data security mode decides what data governance model to use when accessing data
    from a cluster.

    The following modes can only be used when `kind = CLASSIC_PREVIEW`.
    * `DATA_SECURITY_MODE_AUTO`: Databricks will choose the most appropriate access mode depending on your compute configuration.
    * `DATA_SECURITY_MODE_STANDARD`: Alias for `USER_ISOLATION`.
    * `DATA_SECURITY_MODE_DEDICATED`: Alias for `SINGLE_USER`.

    The following modes can be used regardless of `kind`.
    * `NONE`: No security isolation for multiple users sharing the cluster. Data governance features are not available in this mode.
    * `SINGLE_USER`: A secure cluster that can only be exclusively used by a single user specified in `single_user_name`. Most programming languages, cluster features and data governance features are available in this mode.
    * `USER_ISOLATION`: A secure cluster that can be shared by multiple users. Cluster users are fully isolated so that they cannot see each other's data and credentials. Most data governance features are supported in this mode. But programming languages and cluster features might be limited.

    The following modes are deprecated starting with Databricks Runtime 15.0 and
    will be removed for future Databricks Runtime versions:

    * `LEGACY_TABLE_ACL`: This mode is for users migrating from legacy Table ACL clusters.
    * `LEGACY_PASSTHROUGH`: This mode is for users migrating from legacy Passthrough on high concurrency clusters.
    * `LEGACY_SINGLE_USER`: This mode is for users migrating from legacy Passthrough on standard clusters.
    * `LEGACY_SINGLE_USER_STANDARD`: This mode provides a way that doesn’t have UC nor passthrough enabled.
    """

    NONE = "NONE"
    SINGLE_USER = "SINGLE_USER"
    USER_ISOLATION = "USER_ISOLATION"
    LEGACY_TABLE_ACL = "LEGACY_TABLE_ACL"
    LEGACY_PASSTHROUGH = "LEGACY_PASSTHROUGH"
    LEGACY_SINGLE_USER = "LEGACY_SINGLE_USER"
    LEGACY_SINGLE_USER_STANDARD = "LEGACY_SINGLE_USER_STANDARD"
    DATA_SECURITY_MODE_STANDARD = "DATA_SECURITY_MODE_STANDARD"
    DATA_SECURITY_MODE_DEDICATED = "DATA_SECURITY_MODE_DEDICATED"
    DATA_SECURITY_MODE_AUTO = "DATA_SECURITY_MODE_AUTO"


DataSecurityModeParam = (
    Literal[
        "NONE",
        "SINGLE_USER",
        "USER_ISOLATION",
        "LEGACY_TABLE_ACL",
        "LEGACY_PASSTHROUGH",
        "LEGACY_SINGLE_USER",
        "LEGACY_SINGLE_USER_STANDARD",
        "DATA_SECURITY_MODE_STANDARD",
        "DATA_SECURITY_MODE_DEDICATED",
        "DATA_SECURITY_MODE_AUTO",
    ]
    | DataSecurityMode
)
