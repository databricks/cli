from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_permission import (
    AppPermission,
    AppPermissionParam,
)
from databricks.bundles.apps._models.app_resource import AppResource, AppResourceParam
from databricks.bundles.apps._models.compute_size import ComputeSize, ComputeSizeParam
from databricks.bundles.apps._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class App(Resource):
    """"""

    name: VariableOr[str]
    """
    The name of the app. The name must contain only lowercase alphanumeric characters and hyphens.
    It must be unique within the workspace.
    """

    source_code_path: VariableOr[str]

    budget_policy_id: VariableOrOptional[str] = None

    compute_size: VariableOrOptional[ComputeSize] = None

    description: VariableOrOptional[str] = None
    """
    The description of the app.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    permissions: VariableOrList[AppPermission] = field(default_factory=list)

    resources: VariableOrList[AppResource] = field(default_factory=list)
    """
    Resources for the app.
    """

    user_api_scopes: VariableOrList[str] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "AppDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppDict":
        return _transform_to_json_value(self)  # type:ignore


class AppDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    The name of the app. The name must contain only lowercase alphanumeric characters and hyphens.
    It must be unique within the workspace.
    """

    source_code_path: VariableOr[str]

    budget_policy_id: VariableOrOptional[str]

    compute_size: VariableOrOptional[ComputeSizeParam]

    description: VariableOrOptional[str]
    """
    The description of the app.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    permissions: VariableOrList[AppPermissionParam]

    resources: VariableOrList[AppResourceParam]
    """
    Resources for the app.
    """

    user_api_scopes: VariableOrList[str]


AppParam = AppDict | App
