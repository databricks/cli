from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any, TypedDict

from databricks.bundles.apps._models.app_permission import (
    AppPermission,
    AppPermissionParam,
)
from databricks.bundles.apps._models.app_resource import AppResource, AppResourceParam
from databricks.bundles.apps._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class App(Resource):
    """Databricks App resource"""

    name: VariableOr[str]
    """
    The name of the app. The name must be unique within the workspace.
    """

    source_code_path: VariableOrOptional[str] = None
    """
    Path to the app source code on local disk. This is used by DABs to deploy the app.
    """

    description: VariableOrOptional[str] = None
    """
    The description of the app.
    """

    resources: VariableOrList[AppResource] = field(default_factory=list)
    """
    A list of workspace resources associated with the app.
    Each resource can be a job, secret, serving endpoint, SQL warehouse, or Unity Catalog securable.
    """

    permissions: VariableOrList[AppPermission] = field(default_factory=list)
    """
    Access control list for the app. Multiple permissions can be defined for different principals.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    config: VariableOrDict[Any] = field(default_factory=dict)
    """
    Application-specific configuration.

    This can include various settings such as:
    - command: List of strings for the command to run the app
    - env: List of environment variable configurations with 'name', 'value', or 'valueFrom'
    - Any other custom app-specific settings

    See AppConfigDict for common configuration structure.
    """

    @classmethod
    def from_dict(cls, value: "AppDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppDict":
        return _transform_to_json_value(self)  # type:ignore


class AppDict(TypedDict, total=False):
    """Databricks App resource"""

    name: VariableOr[str]
    """
    The name of the app. The name must be unique within the workspace.
    """

    source_code_path: VariableOrOptional[str]
    """
    Path to the app source code on local disk. This is used by DABs to deploy the app.
    """

    description: VariableOrOptional[str]
    """
    The description of the app.
    """

    resources: VariableOrList[AppResourceParam]
    """
    A list of workspace resources associated with the app.
    Each resource can be a job, secret, serving endpoint, SQL warehouse, or Unity Catalog securable.
    """

    permissions: VariableOrList[AppPermissionParam]
    """
    Access control list for the app. Multiple permissions can be defined for different principals.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    config: VariableOrDict[Any]
    """
    Application-specific configuration.

    This can include various settings such as:
    - command: List of strings for the command to run the app
    - env: List of environment variable configurations with 'name', 'value', or 'valueFrom'
    - Any other custom app-specific settings

    See AppConfigDict for common configuration structure.
    """


AppParam = AppDict | App
