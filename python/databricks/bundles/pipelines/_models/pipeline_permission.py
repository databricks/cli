from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.pipelines._models.pipeline_permission_level import (
    PipelinePermissionLevel,
    PipelinePermissionLevelParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelinePermission:
    """"""

    level: VariableOr[PipelinePermissionLevel]
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
    def from_dict(cls, value: "PipelinePermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelinePermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelinePermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[PipelinePermissionLevelParam]
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


PipelinePermissionParam = PipelinePermissionDict | PipelinePermission
