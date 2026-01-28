from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AnthropicConfig:
    """"""

    anthropic_api_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an Anthropic API key. If you
    prefer to paste your API key directly, see `anthropic_api_key_plaintext`.
    You must provide an API key using one of the following fields:
    `anthropic_api_key` or `anthropic_api_key_plaintext`.
    """

    anthropic_api_key_plaintext: VariableOrOptional[str] = None
    """
    The Anthropic API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `anthropic_api_key`. You
    must provide an API key using one of the following fields:
    `anthropic_api_key` or `anthropic_api_key_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "AnthropicConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AnthropicConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class AnthropicConfigDict(TypedDict, total=False):
    """"""

    anthropic_api_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for an Anthropic API key. If you
    prefer to paste your API key directly, see `anthropic_api_key_plaintext`.
    You must provide an API key using one of the following fields:
    `anthropic_api_key` or `anthropic_api_key_plaintext`.
    """

    anthropic_api_key_plaintext: VariableOrOptional[str]
    """
    The Anthropic API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `anthropic_api_key`. You
    must provide an API key using one of the following fields:
    `anthropic_api_key` or `anthropic_api_key_plaintext`.
    """


AnthropicConfigParam = AnthropicConfigDict | AnthropicConfig
