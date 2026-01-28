from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.ai_gateway_guardrail_pii_behavior_behavior import (
    AiGatewayGuardrailPiiBehaviorBehavior,
    AiGatewayGuardrailPiiBehaviorBehaviorParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayGuardrailPiiBehavior:
    """"""

    behavior: VariableOrOptional[AiGatewayGuardrailPiiBehaviorBehavior] = None
    """
    Configuration for input guardrail filters.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayGuardrailPiiBehaviorDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayGuardrailPiiBehaviorDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayGuardrailPiiBehaviorDict(TypedDict, total=False):
    """"""

    behavior: VariableOrOptional[AiGatewayGuardrailPiiBehaviorBehaviorParam]
    """
    Configuration for input guardrail filters.
    """


AiGatewayGuardrailPiiBehaviorParam = (
    AiGatewayGuardrailPiiBehaviorDict | AiGatewayGuardrailPiiBehavior
)
