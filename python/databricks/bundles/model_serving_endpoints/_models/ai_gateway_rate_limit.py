from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.ai_gateway_rate_limit_key import (
    AiGatewayRateLimitKey,
    AiGatewayRateLimitKeyParam,
)
from databricks.bundles.model_serving_endpoints._models.ai_gateway_rate_limit_renewal_period import (
    AiGatewayRateLimitRenewalPeriod,
    AiGatewayRateLimitRenewalPeriodParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayRateLimit:
    """"""

    renewal_period: VariableOr[AiGatewayRateLimitRenewalPeriod]
    """
    Renewal period field for a rate limit. Currently, only 'minute' is supported.
    """

    calls: VariableOrOptional[int] = None
    """
    Used to specify how many calls are allowed for a key within the renewal_period.
    """

    key: VariableOrOptional[AiGatewayRateLimitKey] = None
    """
    Key field for a rate limit. Currently, 'user', 'user_group, 'service_principal', and 'endpoint' are supported,
    with 'endpoint' being the default if not specified.
    """

    principal: VariableOrOptional[str] = None
    """
    Principal field for a user, user group, or service principal to apply rate limiting to. Accepts a user email, group name, or service principal application ID.
    """

    tokens: VariableOrOptional[int] = None
    """
    Used to specify how many tokens are allowed for a key within the renewal_period.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayRateLimitDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayRateLimitDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayRateLimitDict(TypedDict, total=False):
    """"""

    renewal_period: VariableOr[AiGatewayRateLimitRenewalPeriodParam]
    """
    Renewal period field for a rate limit. Currently, only 'minute' is supported.
    """

    calls: VariableOrOptional[int]
    """
    Used to specify how many calls are allowed for a key within the renewal_period.
    """

    key: VariableOrOptional[AiGatewayRateLimitKeyParam]
    """
    Key field for a rate limit. Currently, 'user', 'user_group, 'service_principal', and 'endpoint' are supported,
    with 'endpoint' being the default if not specified.
    """

    principal: VariableOrOptional[str]
    """
    Principal field for a user, user group, or service principal to apply rate limiting to. Accepts a user email, group name, or service principal application ID.
    """

    tokens: VariableOrOptional[int]
    """
    Used to specify how many tokens are allowed for a key within the renewal_period.
    """


AiGatewayRateLimitParam = AiGatewayRateLimitDict | AiGatewayRateLimit
