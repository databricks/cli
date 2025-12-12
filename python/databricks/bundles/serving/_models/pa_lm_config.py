from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PaLmConfig:
    """"""

    palm_api_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a PaLM API key. If you prefer to
    paste your API key directly, see `palm_api_key_plaintext`. You must
    provide an API key using one of the following fields: `palm_api_key` or
    `palm_api_key_plaintext`.
    """

    palm_api_key_plaintext: VariableOrOptional[str] = None
    """
    The PaLM API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `palm_api_key`. You must
    provide an API key using one of the following fields: `palm_api_key` or
    `palm_api_key_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "PaLmConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PaLmConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class PaLmConfigDict(TypedDict, total=False):
    """"""

    palm_api_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for a PaLM API key. If you prefer to
    paste your API key directly, see `palm_api_key_plaintext`. You must
    provide an API key using one of the following fields: `palm_api_key` or
    `palm_api_key_plaintext`.
    """

    palm_api_key_plaintext: VariableOrOptional[str]
    """
    The PaLM API key provided as a plaintext string. If you prefer to
    reference your key using Databricks Secrets, see `palm_api_key`. You must
    provide an API key using one of the following fields: `palm_api_key` or
    `palm_api_key_plaintext`.
    """


PaLmConfigParam = PaLmConfigDict | PaLmConfig
