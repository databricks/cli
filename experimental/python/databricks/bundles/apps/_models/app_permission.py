from dataclasses import dataclass
from enum import Enum
from typing import TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional


class AppPermissionLevel(str, Enum):
    """Permission level for an app"""

    CAN_MANAGE = "CAN_MANAGE"
    CAN_USE = "CAN_USE"


AppPermissionLevelParam = AppPermissionLevel | str


@dataclass(kw_only=True)
class AppPermission:
    """AppPermission holds the permission level setting for a single principal."""

    level: VariableOr[AppPermissionLevel]
    """Permission level"""

    user_name: VariableOrOptional[str] = None
    """User name to grant permission to"""

    service_principal_name: VariableOrOptional[str] = None
    """Service principal name to grant permission to"""

    group_name: VariableOrOptional[str] = None
    """Group name to grant permission to"""

    @classmethod
    def from_dict(cls, value: "AppPermissionDict") -> "AppPermission":
        return _transform(cls, value)

    def as_dict(self) -> "AppPermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class AppPermissionDict(TypedDict, total=False):
    """AppPermission holds the permission level setting for a single principal."""

    level: VariableOr[AppPermissionLevelParam]
    """Permission level"""

    user_name: VariableOrOptional[str]
    """User name to grant permission to"""

    service_principal_name: VariableOrOptional[str]
    """Service principal name to grant permission to"""

    group_name: VariableOrOptional[str]
    """Group name to grant permission to"""


AppPermissionParam = AppPermissionDict | AppPermission
