from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_sql_warehouse_sql_warehouse_permission import (
    AppResourceSqlWarehouseSqlWarehousePermission,
    AppResourceSqlWarehouseSqlWarehousePermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceSqlWarehouse:
    """"""

    id: VariableOr[str]

    permission: VariableOr[AppResourceSqlWarehouseSqlWarehousePermission]

    @classmethod
    def from_dict(cls, value: "AppResourceSqlWarehouseDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceSqlWarehouseDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceSqlWarehouseDict(TypedDict, total=False):
    """"""

    id: VariableOr[str]

    permission: VariableOr[AppResourceSqlWarehouseSqlWarehousePermissionParam]


AppResourceSqlWarehouseParam = AppResourceSqlWarehouseDict | AppResourceSqlWarehouse
