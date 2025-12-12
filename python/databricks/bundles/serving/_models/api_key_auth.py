from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ApiKeyAuth:
    """"""

    key: VariableOr[str]
    """
    The name of the API key parameter used for authentication.
    """

    value: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an API Key.
    If you prefer to paste your token directly, see `value_plaintext`.
    """

    value_plaintext: VariableOrOptional[str] = None
    """
    The API Key provided as a plaintext string. If you prefer to reference your
    token using Databricks Secrets, see `value`.
    """

    @classmethod
    def from_dict(cls, value: "ApiKeyAuthDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ApiKeyAuthDict":
        return _transform_to_json_value(self)  # type:ignore


class ApiKeyAuthDict(TypedDict, total=False):
    """"""

    key: VariableOr[str]
    """
    The name of the API key parameter used for authentication.
    """

    value: VariableOrOptional[str]
    """
    The Databricks secret key reference for an API Key.
    If you prefer to paste your token directly, see `value_plaintext`.
    """

    value_plaintext: VariableOrOptional[str]
    """
    The API Key provided as a plaintext string. If you prefer to reference your
    token using Databricks Secrets, see `value`.
    """


ApiKeyAuthParam = ApiKeyAuthDict | ApiKeyAuth
