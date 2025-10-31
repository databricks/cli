from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_database import (
    AppResourceDatabase,
    AppResourceDatabaseParam,
)
from databricks.bundles.apps._models.app_resource_genie_space import (
    AppResourceGenieSpace,
    AppResourceGenieSpaceParam,
)
from databricks.bundles.apps._models.app_resource_job import (
    AppResourceJob,
    AppResourceJobParam,
)
from databricks.bundles.apps._models.app_resource_secret import (
    AppResourceSecret,
    AppResourceSecretParam,
)
from databricks.bundles.apps._models.app_resource_serving_endpoint import (
    AppResourceServingEndpoint,
    AppResourceServingEndpointParam,
)
from databricks.bundles.apps._models.app_resource_sql_warehouse import (
    AppResourceSqlWarehouse,
    AppResourceSqlWarehouseParam,
)
from databricks.bundles.apps._models.app_resource_uc_securable import (
    AppResourceUcSecurable,
    AppResourceUcSecurableParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResource:
    """"""

    name: VariableOr[str]
    """
    Name of the App Resource.
    """

    database: VariableOrOptional[AppResourceDatabase] = None

    description: VariableOrOptional[str] = None
    """
    Description of the App Resource.
    """

    genie_space: VariableOrOptional[AppResourceGenieSpace] = None

    job: VariableOrOptional[AppResourceJob] = None

    secret: VariableOrOptional[AppResourceSecret] = None

    serving_endpoint: VariableOrOptional[AppResourceServingEndpoint] = None

    sql_warehouse: VariableOrOptional[AppResourceSqlWarehouse] = None

    uc_securable: VariableOrOptional[AppResourceUcSecurable] = None

    @classmethod
    def from_dict(cls, value: "AppResourceDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    Name of the App Resource.
    """

    database: VariableOrOptional[AppResourceDatabaseParam]

    description: VariableOrOptional[str]
    """
    Description of the App Resource.
    """

    genie_space: VariableOrOptional[AppResourceGenieSpaceParam]

    job: VariableOrOptional[AppResourceJobParam]

    secret: VariableOrOptional[AppResourceSecretParam]

    serving_endpoint: VariableOrOptional[AppResourceServingEndpointParam]

    sql_warehouse: VariableOrOptional[AppResourceSqlWarehouseParam]

    uc_securable: VariableOrOptional[AppResourceUcSecurableParam]


AppResourceParam = AppResourceDict | AppResource
