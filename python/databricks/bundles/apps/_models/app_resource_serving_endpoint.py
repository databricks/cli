from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_serving_endpoint_serving_endpoint_permission import (
    AppResourceServingEndpointServingEndpointPermission,
    AppResourceServingEndpointServingEndpointPermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceServingEndpoint:
    """"""

    name: VariableOr[str]

    permission: VariableOr[AppResourceServingEndpointServingEndpointPermission]

    @classmethod
    def from_dict(cls, value: "AppResourceServingEndpointDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceServingEndpointDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceServingEndpointDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]

    permission: VariableOr[AppResourceServingEndpointServingEndpointPermissionParam]


AppResourceServingEndpointParam = (
    AppResourceServingEndpointDict | AppResourceServingEndpoint
)
