from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.model_serving_endpoint_permission_level import (
    ModelServingEndpointPermissionLevel,
    ModelServingEndpointPermissionLevelParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ModelServingEndpointPermission:
    """"""

    level: VariableOr[ModelServingEndpointPermissionLevel]

    group_name: VariableOrOptional[str] = None

    service_principal_name: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "ModelServingEndpointPermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ModelServingEndpointPermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class ModelServingEndpointPermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[ModelServingEndpointPermissionLevelParam]

    group_name: VariableOrOptional[str]

    service_principal_name: VariableOrOptional[str]

    user_name: VariableOrOptional[str]


ModelServingEndpointPermissionParam = (
    ModelServingEndpointPermissionDict | ModelServingEndpointPermission
)
