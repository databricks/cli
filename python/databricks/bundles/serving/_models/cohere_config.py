from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CohereConfig:
    """"""

    cohere_api_base: VariableOrOptional[str] = None
    """
    This is an optional field to provide a customized base URL for the Cohere
    API. If left unspecified, the standard Cohere base URL is used.
    """

    cohere_api_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a Cohere API key. If you prefer
    to paste your API key directly, see `cohere_api_key_plaintext`. You must
    provide an API key using one of the following fields: `cohere_api_key` or
    `cohere_api_key_plaintext`.
    """

    cohere_api_key_plaintext: VariableOrOptional[str] = None
    """
    The Cohere API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `cohere_api_key`. You
    must provide an API key using one of the following fields:
    `cohere_api_key` or `cohere_api_key_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "CohereConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CohereConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class CohereConfigDict(TypedDict, total=False):
    """"""

    cohere_api_base: VariableOrOptional[str]
    """
    This is an optional field to provide a customized base URL for the Cohere
    API. If left unspecified, the standard Cohere base URL is used.
    """

    cohere_api_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for a Cohere API key. If you prefer
    to paste your API key directly, see `cohere_api_key_plaintext`. You must
    provide an API key using one of the following fields: `cohere_api_key` or
    `cohere_api_key_plaintext`.
    """

    cohere_api_key_plaintext: VariableOrOptional[str]
    """
    The Cohere API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `cohere_api_key`. You
    must provide an API key using one of the following fields:
    `cohere_api_key` or `cohere_api_key_plaintext`.
    """


CohereConfigParam = CohereConfigDict | CohereConfig
