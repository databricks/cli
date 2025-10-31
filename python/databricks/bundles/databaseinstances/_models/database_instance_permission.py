from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.databaseinstances._models.database_instance_permission_level import (
    DatabaseInstancePermissionLevel,
    DatabaseInstancePermissionLevelParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DatabaseInstancePermission:
    """"""

    level: VariableOr[DatabaseInstancePermissionLevel]

    group_name: VariableOrOptional[str] = None

    service_principal_name: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "DatabaseInstancePermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DatabaseInstancePermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class DatabaseInstancePermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[DatabaseInstancePermissionLevelParam]

    group_name: VariableOrOptional[str]

    service_principal_name: VariableOrOptional[str]

    user_name: VariableOrOptional[str]


DatabaseInstancePermissionParam = (
    DatabaseInstancePermissionDict | DatabaseInstancePermission
)
