from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.ai_gateway_guardrail_pii_behavior import (
    AiGatewayGuardrailPiiBehavior,
    AiGatewayGuardrailPiiBehaviorParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayGuardrailParameters:
    """"""

    invalid_keywords: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] List of invalid keywords.
    AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
    """

    pii: VariableOrOptional[AiGatewayGuardrailPiiBehavior] = None
    """
    Configuration for guardrail PII filter.
    """

    safety: VariableOrOptional[bool] = None
    """
    Indicates whether the safety filter is enabled.
    """

    valid_topics: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] The list of allowed topics.
    Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayGuardrailParametersDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayGuardrailParametersDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayGuardrailParametersDict(TypedDict, total=False):
    """"""

    invalid_keywords: VariableOrList[str]
    """
    [DEPRECATED] List of invalid keywords.
    AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
    """

    pii: VariableOrOptional[AiGatewayGuardrailPiiBehaviorParam]
    """
    Configuration for guardrail PII filter.
    """

    safety: VariableOrOptional[bool]
    """
    Indicates whether the safety filter is enabled.
    """

    valid_topics: VariableOrList[str]
    """
    [DEPRECATED] The list of allowed topics.
    Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
    """


AiGatewayGuardrailParametersParam = (
    AiGatewayGuardrailParametersDict | AiGatewayGuardrailParameters
)
