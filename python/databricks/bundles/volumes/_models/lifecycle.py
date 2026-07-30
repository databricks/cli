from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.volumes._models.privilege_assignment import PrivilegeAssignment, PrivilegeAssignmentDict, PrivilegeAssignmentParam
from databricks.bundles.volumes._models.volume import Volume, VolumeDict, VolumeParam
from databricks.bundles.volumes._models.privilege import Privilege, PrivilegeParam

from databricks.bundles.volumes._models.volume_type import VolumeType, VolumeTypeParam


if TYPE_CHECKING:
    from typing_extensions import Self

@dataclass(kw_only=True)
class Lifecycle:
    """"""

    prevent_destroy: VariableOrOptional[bool] = None
    """
    Lifecycle setting to prevent the resource from being destroyed.
    """

    @classmethod
    def from_dict(cls, value: 'LifecycleDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'LifecycleDict':
        return _transform_to_json_value(self) # type:ignore



class LifecycleDict(TypedDict, total=False):
    """"""

    prevent_destroy: VariableOrOptional[bool]
    """
    Lifecycle setting to prevent the resource from being destroyed.
    """


LifecycleParam = LifecycleDict | Lifecycle
