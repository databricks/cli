from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.rate_limit_key import (
    RateLimitKey,
    RateLimitKeyParam,
)
from databricks.bundles.model_serving_endpoints._models.rate_limit_renewal_period import (
    RateLimitRenewalPeriod,
    RateLimitRenewalPeriodParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class RateLimit:
    """
    [DEPRECATED]
    """

    calls: VariableOr[int]
    """
    Used to specify how many calls are allowed for a key within the renewal_period.
    """

    renewal_period: VariableOr[RateLimitRenewalPeriod]
    """
    Renewal period field for a serving endpoint rate limit. Currently, only 'minute' is supported.
    """

    key: VariableOrOptional[RateLimitKey] = None
    """
    Key field for a serving endpoint rate limit. Currently, only 'user' and 'endpoint' are supported, with 'endpoint' being the default if not specified.
    """

    @classmethod
    def from_dict(cls, value: "RateLimitDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RateLimitDict":
        return _transform_to_json_value(self)  # type:ignore


class RateLimitDict(TypedDict, total=False):
    """"""

    calls: VariableOr[int]
    """
    Used to specify how many calls are allowed for a key within the renewal_period.
    """

    renewal_period: VariableOr[RateLimitRenewalPeriodParam]
    """
    Renewal period field for a serving endpoint rate limit. Currently, only 'minute' is supported.
    """

    key: VariableOrOptional[RateLimitKeyParam]
    """
    Key field for a serving endpoint rate limit. Currently, only 'user' and 'endpoint' are supported, with 'endpoint' being the default if not specified.
    """


RateLimitParam = RateLimitDict | RateLimit
