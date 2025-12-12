from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.serving._models.ai_gateway_guardrail_parameters import (
    AiGatewayGuardrailParameters,
    AiGatewayGuardrailParametersParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayGuardrails:
    """"""

    input: VariableOrOptional[AiGatewayGuardrailParameters] = None
    """
    Configuration for input guardrail filters.
    """

    output: VariableOrOptional[AiGatewayGuardrailParameters] = None
    """
    Configuration for output guardrail filters.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayGuardrailsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayGuardrailsDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayGuardrailsDict(TypedDict, total=False):
    """"""

    input: VariableOrOptional[AiGatewayGuardrailParametersParam]
    """
    Configuration for input guardrail filters.
    """

    output: VariableOrOptional[AiGatewayGuardrailParametersParam]
    """
    Configuration for output guardrail filters.
    """


AiGatewayGuardrailsParam = AiGatewayGuardrailsDict | AiGatewayGuardrails
