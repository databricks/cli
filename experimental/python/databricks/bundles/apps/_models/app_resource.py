from dataclasses import dataclass
from enum import Enum
from typing import TYPE_CHECKING, TypedDict, Union

from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


class JobPermission(str, Enum):
    """Permission for job resources"""

    JOB_RUN = "JOB_RUN"


class SecretPermission(str, Enum):
    """Permission for secret resources"""

    READ = "READ"
    WRITE = "WRITE"


class ServingEndpointPermission(str, Enum):
    """Permission for serving endpoint resources"""

    CAN_QUERY = "CAN_QUERY"
    CAN_MANAGE = "CAN_MANAGE"


class SqlWarehousePermission(str, Enum):
    """Permission for SQL warehouse resources"""

    CAN_USE = "CAN_USE"


class UcSecurableType(str, Enum):
    """Type of Unity Catalog securable"""

    VOLUME = "VOLUME"


class UcSecurablePermission(str, Enum):
    """Permission for Unity Catalog securable resources"""

    READ_VOLUME = "READ_VOLUME"
    WRITE_VOLUME = "WRITE_VOLUME"


@dataclass(kw_only=True)
class AppResourceJob:
    """Reference to a job resource"""

    id: VariableOr[str]
    """The ID of the job"""

    permission: VariableOr[JobPermission | str]
    """The permission to grant on the job"""


class AppResourceJobDict(TypedDict, total=False):
    """Reference to a job resource"""

    id: VariableOr[str]
    """The ID of the job"""

    permission: VariableOr[JobPermission | str]
    """The permission to grant on the job"""


AppResourceJobParam = AppResourceJobDict | AppResourceJob


@dataclass(kw_only=True)
class AppResourceSecret:
    """Reference to a secret resource"""

    key: VariableOr[str]
    """The key of the secret"""

    scope: VariableOr[str]
    """The scope of the secret"""

    permission: VariableOr[SecretPermission | str]
    """The permission to grant on the secret"""


class AppResourceSecretDict(TypedDict, total=False):
    """Reference to a secret resource"""

    key: VariableOr[str]
    """The key of the secret"""

    scope: VariableOr[str]
    """The scope of the secret"""

    permission: VariableOr[SecretPermission | str]
    """The permission to grant on the secret"""


AppResourceSecretParam = AppResourceSecretDict | AppResourceSecret


@dataclass(kw_only=True)
class AppResourceServingEndpoint:
    """Reference to a serving endpoint resource"""

    name: VariableOr[str]
    """The name of the serving endpoint"""

    permission: VariableOr[ServingEndpointPermission | str]
    """The permission to grant on the serving endpoint"""


class AppResourceServingEndpointDict(TypedDict, total=False):
    """Reference to a serving endpoint resource"""

    name: VariableOr[str]
    """The name of the serving endpoint"""

    permission: VariableOr[ServingEndpointPermission | str]
    """The permission to grant on the serving endpoint"""


AppResourceServingEndpointParam = (
    AppResourceServingEndpointDict | AppResourceServingEndpoint
)


@dataclass(kw_only=True)
class AppResourceSqlWarehouse:
    """Reference to a SQL warehouse resource"""

    id: VariableOr[str]
    """The ID of the SQL warehouse"""

    permission: VariableOr[SqlWarehousePermission | str]
    """The permission to grant on the SQL warehouse"""


class AppResourceSqlWarehouseDict(TypedDict, total=False):
    """Reference to a SQL warehouse resource"""

    id: VariableOr[str]
    """The ID of the SQL warehouse"""

    permission: VariableOr[SqlWarehousePermission | str]
    """The permission to grant on the SQL warehouse"""


AppResourceSqlWarehouseParam = AppResourceSqlWarehouseDict | AppResourceSqlWarehouse


@dataclass(kw_only=True)
class AppResourceUcSecurable:
    """Reference to a Unity Catalog securable resource"""

    name: VariableOr[str]
    """The name of the securable"""

    securable_type: VariableOr[UcSecurableType | str]
    """The type of the securable"""

    permission: VariableOr[UcSecurablePermission | str]
    """The permission to grant on the securable"""


class AppResourceUcSecurableDict(TypedDict, total=False):
    """Reference to a Unity Catalog securable resource"""

    name: VariableOr[str]
    """The name of the securable"""

    securable_type: VariableOr[UcSecurableType | str]
    """The type of the securable"""

    permission: VariableOr[UcSecurablePermission | str]
    """The permission to grant on the securable"""


AppResourceUcSecurableParam = AppResourceUcSecurableDict | AppResourceUcSecurable


# Union of all app resource types
AppResource = Union[
    AppResourceJob,
    AppResourceSecret,
    AppResourceServingEndpoint,
    AppResourceSqlWarehouse,
    AppResourceUcSecurable,
]

AppResourceParam = Union[
    AppResourceJobParam,
    AppResourceSecretParam,
    AppResourceServingEndpointParam,
    AppResourceSqlWarehouseParam,
    AppResourceUcSecurableParam,
]
