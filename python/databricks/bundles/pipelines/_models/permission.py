from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Permission:
    """"""

    level: VariableOr[str]
    """
    The allowed permission for user, group, service principal defined for this permission.
    """

    group_name: VariableOrOptional[str] = None
    """
    The name of the group that has the permission set in level.
    """

    service_principal_name: VariableOrOptional[str] = None
    """
    The name of the service principal that has the permission set in level.
    """

    user_name: VariableOrOptional[str] = None
    """
    The name of the user that has the permission set in level.
    """

    @classmethod
    def from_dict(cls, value: "PermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class PermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[str]
    """
    The allowed permission for user, group, service principal defined for this permission.
    """

    group_name: VariableOrOptional[str]
    """
    The name of the group that has the permission set in level.
    """

    service_principal_name: VariableOrOptional[str]
    """
    The name of the service principal that has the permission set in level.
    """

    user_name: VariableOrOptional[str]
    """
    The name of the user that has the permission set in level.
    """


PermissionParam = PermissionDict | Permission
