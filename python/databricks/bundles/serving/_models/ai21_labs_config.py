from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Ai21LabsConfig:
    """"""

    ai21labs_api_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an AI21 Labs API key. If you
    prefer to paste your API key directly, see `ai21labs_api_key_plaintext`.
    You must provide an API key using one of the following fields:
    `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
    """

    ai21labs_api_key_plaintext: VariableOrOptional[str] = None
    """
    An AI21 Labs API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `ai21labs_api_key`. You
    must provide an API key using one of the following fields:
    `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "Ai21LabsConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "Ai21LabsConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class Ai21LabsConfigDict(TypedDict, total=False):
    """"""

    ai21labs_api_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for an AI21 Labs API key. If you
    prefer to paste your API key directly, see `ai21labs_api_key_plaintext`.
    You must provide an API key using one of the following fields:
    `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
    """

    ai21labs_api_key_plaintext: VariableOrOptional[str]
    """
    An AI21 Labs API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `ai21labs_api_key`. You
    must provide an API key using one of the following fields:
    `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
    """


Ai21LabsConfigParam = Ai21LabsConfigDict | Ai21LabsConfig
