from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class BearerTokenAuth:
    """"""

    token: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a token.
    If you prefer to paste your token directly, see `token_plaintext`.
    """

    token_plaintext: VariableOrOptional[str] = None
    """
    The token provided as a plaintext string. If you prefer to reference your
    token using Databricks Secrets, see `token`.
    """

    @classmethod
    def from_dict(cls, value: "BearerTokenAuthDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "BearerTokenAuthDict":
        return _transform_to_json_value(self)  # type:ignore


class BearerTokenAuthDict(TypedDict, total=False):
    """"""

    token: VariableOrOptional[str]
    """
    The Databricks secret key reference for a token.
    If you prefer to paste your token directly, see `token_plaintext`.
    """

    token_plaintext: VariableOrOptional[str]
    """
    The token provided as a plaintext string. If you prefer to reference your
    token using Databricks Secrets, see `token`.
    """


BearerTokenAuthParam = BearerTokenAuthDict | BearerTokenAuth
