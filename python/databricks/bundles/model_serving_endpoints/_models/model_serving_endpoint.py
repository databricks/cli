from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.model_serving_endpoints._models.ai_gateway_config import (
    AiGatewayConfig,
    AiGatewayConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.email_notifications import (
    EmailNotifications,
    EmailNotificationsParam,
)
from databricks.bundles.model_serving_endpoints._models.endpoint_core_config_input import (
    EndpointCoreConfigInput,
    EndpointCoreConfigInputParam,
)
from databricks.bundles.model_serving_endpoints._models.endpoint_tag import (
    EndpointTag,
    EndpointTagParam,
)
from databricks.bundles.model_serving_endpoints._models.lifecycle import (
    Lifecycle,
    LifecycleParam,
)
from databricks.bundles.model_serving_endpoints._models.model_serving_endpoint_permission import (
    ModelServingEndpointPermission,
    ModelServingEndpointPermissionParam,
)
from databricks.bundles.model_serving_endpoints._models.rate_limit import (
    RateLimit,
    RateLimitParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ModelServingEndpoint(Resource):
    """"""

    name: VariableOr[str]
    """
    The name of the serving endpoint. This field is required and must be unique across a Databricks workspace.
    An endpoint name can consist of alphanumeric characters, dashes, and underscores.
    """

    ai_gateway: VariableOrOptional[AiGatewayConfig] = None
    """
    The AI Gateway configuration for the serving endpoint. NOTE: External model, provisioned throughput, and pay-per-token endpoints are fully supported; agent endpoints currently only support inference tables.
    """

    budget_policy_id: VariableOrOptional[str] = None
    """
    The budget policy to be applied to the serving endpoint.
    """

    config: VariableOrOptional[EndpointCoreConfigInput] = None
    """
    The core config of the serving endpoint.
    """

    description: VariableOrOptional[str] = None

    email_notifications: VariableOrOptional[EmailNotifications] = None
    """
    Email notification settings.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    permissions: VariableOrList[ModelServingEndpointPermission] = field(
        default_factory=list
    )

    rate_limits: VariableOrList[RateLimit] = field(default_factory=list)
    """
    [DEPRECATED] Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
    """

    route_optimized: VariableOrOptional[bool] = None
    """
    Enable route optimization for the serving endpoint.
    """

    tags: VariableOrList[EndpointTag] = field(default_factory=list)
    """
    Tags to be attached to the serving endpoint and automatically propagated to billing logs.
    """

    @classmethod
    def from_dict(cls, value: "ModelServingEndpointDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ModelServingEndpointDict":
        return _transform_to_json_value(self)  # type:ignore


class ModelServingEndpointDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    The name of the serving endpoint. This field is required and must be unique across a Databricks workspace.
    An endpoint name can consist of alphanumeric characters, dashes, and underscores.
    """

    ai_gateway: VariableOrOptional[AiGatewayConfigParam]
    """
    The AI Gateway configuration for the serving endpoint. NOTE: External model, provisioned throughput, and pay-per-token endpoints are fully supported; agent endpoints currently only support inference tables.
    """

    budget_policy_id: VariableOrOptional[str]
    """
    The budget policy to be applied to the serving endpoint.
    """

    config: VariableOrOptional[EndpointCoreConfigInputParam]
    """
    The core config of the serving endpoint.
    """

    description: VariableOrOptional[str]

    email_notifications: VariableOrOptional[EmailNotificationsParam]
    """
    Email notification settings.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    permissions: VariableOrList[ModelServingEndpointPermissionParam]

    rate_limits: VariableOrList[RateLimitParam]
    """
    [DEPRECATED] Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
    """

    route_optimized: VariableOrOptional[bool]
    """
    Enable route optimization for the serving endpoint.
    """

    tags: VariableOrList[EndpointTagParam]
    """
    Tags to be attached to the serving endpoint and automatically propagated to billing logs.
    """


ModelServingEndpointParam = ModelServingEndpointDict | ModelServingEndpoint
