from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.ai_gateway_guardrails import (
    AiGatewayGuardrails,
    AiGatewayGuardrailsParam,
)
from databricks.bundles.model_serving_endpoints._models.ai_gateway_inference_table_config import (
    AiGatewayInferenceTableConfig,
    AiGatewayInferenceTableConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.ai_gateway_rate_limit import (
    AiGatewayRateLimit,
    AiGatewayRateLimitParam,
)
from databricks.bundles.model_serving_endpoints._models.ai_gateway_usage_tracking_config import (
    AiGatewayUsageTrackingConfig,
    AiGatewayUsageTrackingConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.fallback_config import (
    FallbackConfig,
    FallbackConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayConfig:
    """"""

    fallback_config: VariableOrOptional[FallbackConfig] = None
    """
    Configuration for traffic fallback which auto fallbacks to other served entities if the request to a served
    entity fails with certain error codes, to increase availability.
    """

    guardrails: VariableOrOptional[AiGatewayGuardrails] = None
    """
    Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
    """

    inference_table_config: VariableOrOptional[AiGatewayInferenceTableConfig] = None
    """
    Configuration for payload logging using inference tables.
    Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
    """

    rate_limits: VariableOrList[AiGatewayRateLimit] = field(default_factory=list)
    """
    Configuration for rate limits which can be set to limit endpoint traffic.
    """

    usage_tracking_config: VariableOrOptional[AiGatewayUsageTrackingConfig] = None
    """
    Configuration to enable usage tracking using system tables.
    These tables allow you to monitor operational usage on endpoints and their associated costs.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayConfigDict(TypedDict, total=False):
    """"""

    fallback_config: VariableOrOptional[FallbackConfigParam]
    """
    Configuration for traffic fallback which auto fallbacks to other served entities if the request to a served
    entity fails with certain error codes, to increase availability.
    """

    guardrails: VariableOrOptional[AiGatewayGuardrailsParam]
    """
    Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
    """

    inference_table_config: VariableOrOptional[AiGatewayInferenceTableConfigParam]
    """
    Configuration for payload logging using inference tables.
    Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
    """

    rate_limits: VariableOrList[AiGatewayRateLimitParam]
    """
    Configuration for rate limits which can be set to limit endpoint traffic.
    """

    usage_tracking_config: VariableOrOptional[AiGatewayUsageTrackingConfigParam]
    """
    Configuration to enable usage tracking using system tables.
    These tables allow you to monitor operational usage on endpoints and their associated costs.
    """


AiGatewayConfigParam = AiGatewayConfigDict | AiGatewayConfig
