from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_database_database_permission import (
    AppResourceDatabaseDatabasePermission,
    AppResourceDatabaseDatabasePermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceDatabase:
    """"""

    database_name: VariableOr[str]

    instance_name: VariableOr[str]

    permission: VariableOr[AppResourceDatabaseDatabasePermission]

    @classmethod
    def from_dict(cls, value: "AppResourceDatabaseDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceDatabaseDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceDatabaseDict(TypedDict, total=False):
    """"""

    database_name: VariableOr[str]

    instance_name: VariableOr[str]

    permission: VariableOr[AppResourceDatabaseDatabasePermissionParam]


AppResourceDatabaseParam = AppResourceDatabaseDict | AppResourceDatabase
