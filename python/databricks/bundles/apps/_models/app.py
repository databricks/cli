from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_deployment import (
    AppDeployment,
    AppDeploymentParam,
)
from databricks.bundles.apps._models.app_permission import (
    AppPermission,
    AppPermissionParam,
)
from databricks.bundles.apps._models.app_resource import AppResource, AppResourceParam
from databricks.bundles.apps._models.application_status import (
    ApplicationStatus,
    ApplicationStatusParam,
)
from databricks.bundles.apps._models.compute_size import ComputeSize, ComputeSizeParam
from databricks.bundles.apps._models.compute_status import (
    ComputeStatus,
    ComputeStatusParam,
)
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

    active_deployment: VariableOrOptional[AppDeployment] = None
    """
    The active deployment of the app. A deployment is considered active when it has been deployed
    to the app compute.
    """

    app_status: VariableOrOptional[ApplicationStatus] = None

    budget_policy_id: VariableOrOptional[str] = None

    compute_size: VariableOrOptional[ComputeSize] = None

    compute_status: VariableOrOptional[ComputeStatus] = None

    create_time: VariableOrOptional[str] = None
    """
    The creation time of the app. Formatted timestamp in ISO 6801.
    """

    creator: VariableOrOptional[str] = None
    """
    The email of the user that created the app.
    """

    default_source_code_path: VariableOrOptional[str] = None
    """
    The default workspace file system path of the source code from which app deployment are
    created. This field tracks the workspace source code path of the last active deployment.
    """

    description: VariableOrOptional[str] = None
    """
    The description of the app.
    """

    effective_budget_policy_id: VariableOrOptional[str] = None

    effective_user_api_scopes: VariableOrList[str] = field(default_factory=list)
    """
    The effective api scopes granted to the user access token.
    """

    id: VariableOrOptional[str] = None
    """
    The unique identifier of the app.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    oauth2_app_client_id: VariableOrOptional[str] = None

    oauth2_app_integration_id: VariableOrOptional[str] = None

    pending_deployment: VariableOrOptional[AppDeployment] = None
    """
    The pending deployment of the app. A deployment is considered pending when it is being prepared
    for deployment to the app compute.
    """

    permissions: VariableOrList[AppPermission] = field(default_factory=list)

    resources: VariableOrList[AppResource] = field(default_factory=list)
    """
    Resources for the app.
    """

    service_principal_client_id: VariableOrOptional[str] = None

    service_principal_id: VariableOrOptional[int] = None

    service_principal_name: VariableOrOptional[str] = None

    update_time: VariableOrOptional[str] = None
    """
    The update time of the app. Formatted timestamp in ISO 6801.
    """

    updater: VariableOrOptional[str] = None
    """
    The email of the user that last updated the app.
    """

    url: VariableOrOptional[str] = None
    """
    The URL of the app once it is deployed.
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

    active_deployment: VariableOrOptional[AppDeploymentParam]
    """
    The active deployment of the app. A deployment is considered active when it has been deployed
    to the app compute.
    """

    app_status: VariableOrOptional[ApplicationStatusParam]

    budget_policy_id: VariableOrOptional[str]

    compute_size: VariableOrOptional[ComputeSizeParam]

    compute_status: VariableOrOptional[ComputeStatusParam]

    create_time: VariableOrOptional[str]
    """
    The creation time of the app. Formatted timestamp in ISO 6801.
    """

    creator: VariableOrOptional[str]
    """
    The email of the user that created the app.
    """

    default_source_code_path: VariableOrOptional[str]
    """
    The default workspace file system path of the source code from which app deployment are
    created. This field tracks the workspace source code path of the last active deployment.
    """

    description: VariableOrOptional[str]
    """
    The description of the app.
    """

    effective_budget_policy_id: VariableOrOptional[str]

    effective_user_api_scopes: VariableOrList[str]
    """
    The effective api scopes granted to the user access token.
    """

    id: VariableOrOptional[str]
    """
    The unique identifier of the app.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    oauth2_app_client_id: VariableOrOptional[str]

    oauth2_app_integration_id: VariableOrOptional[str]

    pending_deployment: VariableOrOptional[AppDeploymentParam]
    """
    The pending deployment of the app. A deployment is considered pending when it is being prepared
    for deployment to the app compute.
    """

    permissions: VariableOrList[AppPermissionParam]

    resources: VariableOrList[AppResourceParam]
    """
    Resources for the app.
    """

    service_principal_client_id: VariableOrOptional[str]

    service_principal_id: VariableOrOptional[int]

    service_principal_name: VariableOrOptional[str]

    update_time: VariableOrOptional[str]
    """
    The update time of the app. Formatted timestamp in ISO 6801.
    """

    updater: VariableOrOptional[str]
    """
    The email of the user that last updated the app.
    """

    url: VariableOrOptional[str]
    """
    The URL of the app once it is deployed.
    """

    user_api_scopes: VariableOrList[str]


AppParam = AppDict | App
