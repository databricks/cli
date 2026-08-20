from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.alerts._models.permission_level import (
    PermissionLevel,
    PermissionLevelParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Permission:
    """"""

    level: VariableOr[PermissionLevel]
    """
    The permission level to apply. The allowed levels depend on the resource type.
    """

    group_name: VariableOrOptional[str] = None
    """
    The name of the group granted the permission level.
    """

    service_principal_name: VariableOrOptional[str] = None
    """
    The name of the service principal granted the permission level.
    """

    user_name: VariableOrOptional[str] = None
    """
    The name of the user granted the permission level.
    """

    @classmethod
    def from_dict(cls, value: "PermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class PermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[PermissionLevelParam]
    """
    The permission level to apply. The allowed levels depend on the resource type.
    """

    group_name: VariableOrOptional[str]
    """
    The name of the group granted the permission level.
    """

    service_principal_name: VariableOrOptional[str]
    """
    The name of the service principal granted the permission level.
    """

    user_name: VariableOrOptional[str]
    """
    The name of the user granted the permission level.
    """


PermissionParam = PermissionDict | Permission
