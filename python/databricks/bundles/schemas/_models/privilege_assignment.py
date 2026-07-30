from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.schemas._models.lifecycle import Lifecycle, LifecycleDict, LifecycleParam
from databricks.bundles.schemas._models.schema import Schema, SchemaDict, SchemaParam
from databricks.bundles.schemas._models.privilege import Privilege, PrivilegeParam


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
    def from_dict(cls, value: 'PrivilegeAssignmentDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'PrivilegeAssignmentDict':
        return _transform_to_json_value(self) # type:ignore



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
