from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class FallbackConfig:
    """"""

    enabled: VariableOr[bool]
    """
    Whether to enable traffic fallback. When a served entity in the serving endpoint returns specific error
    codes (e.g. 500), the request will automatically be round-robin attempted with other served entities in the same
    endpoint, following the order of served entity list, until a successful response is returned.
    If all attempts fail, return the last response with the error code.
    """

    @classmethod
    def from_dict(cls, value: "FallbackConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FallbackConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class FallbackConfigDict(TypedDict, total=False):
    """"""

    enabled: VariableOr[bool]
    """
    Whether to enable traffic fallback. When a served entity in the serving endpoint returns specific error
    codes (e.g. 500), the request will automatically be round-robin attempted with other served entities in the same
    endpoint, following the order of served entity list, until a successful response is returned.
    If all attempts fail, return the last response with the error code.
    """


FallbackConfigParam = FallbackConfigDict | FallbackConfig
