from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayUsageTrackingConfig:
    """"""

    enabled: VariableOrOptional[bool] = None
    """
    Whether to enable usage tracking.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayUsageTrackingConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayUsageTrackingConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayUsageTrackingConfigDict(TypedDict, total=False):
    """"""

    enabled: VariableOrOptional[bool]
    """
    Whether to enable usage tracking.
    """


AiGatewayUsageTrackingConfigParam = (
    AiGatewayUsageTrackingConfigDict | AiGatewayUsageTrackingConfig
)
