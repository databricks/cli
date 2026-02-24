from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SparseCheckout:
    """
    :meta private: [EXPERIMENTAL]
    """

    patterns: VariableOrList[str] = field(default_factory=list)
    """
    List of patterns to include for sparse checkout.
    """

    @classmethod
    def from_dict(cls, value: "SparseCheckoutDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SparseCheckoutDict":
        return _transform_to_json_value(self)  # type:ignore


class SparseCheckoutDict(TypedDict, total=False):
    """"""

    patterns: VariableOrList[str]
    """
    List of patterns to include for sparse checkout.
    """


SparseCheckoutParam = SparseCheckoutDict | SparseCheckout
