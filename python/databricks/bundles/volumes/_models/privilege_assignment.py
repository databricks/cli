from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.volumes._models.privilege import Privilege, PrivilegeParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PrivilegeAssignment:
    """"""

    principal: VariableOrOptional[str] = None
    """
    The principal (user email address or group name).
    For deleted principals, `principal` is empty while `principal_id` is populated.
    """

    privileges: VariableOrList[Privilege] = field(default_factory=list)
    """
    The privileges assigned to the principal.
    """

    @classmethod
    def from_dict(cls, value: "PrivilegeAssignmentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PrivilegeAssignmentDict":
        return _transform_to_json_value(self)  # type:ignore


class PrivilegeAssignmentDict(TypedDict, total=False):
    """"""

    principal: VariableOrOptional[str]
    """
    The principal (user email address or group name).
    For deleted principals, `principal` is empty while `principal_id` is populated.
    """

    privileges: VariableOrList[PrivilegeParam]
    """
    The privileges assigned to the principal.
    """


PrivilegeAssignmentParam = PrivilegeAssignmentDict | PrivilegeAssignment
