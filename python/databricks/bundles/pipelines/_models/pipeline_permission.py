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

    group_name: VariableOrOptional[str] = None

    service_principal_name: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "PipelinePermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelinePermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelinePermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[PipelinePermissionLevelParam]

    group_name: VariableOrOptional[str]

    service_principal_name: VariableOrOptional[str]

    user_name: VariableOrOptional[str]


PipelinePermissionParam = PipelinePermissionDict | PipelinePermission
