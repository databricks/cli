from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Filters:
    """"""

    exclude: VariableOrList[str] = field(default_factory=list)
    """
    Paths to exclude.
    """

    include: VariableOrList[str] = field(default_factory=list)
    """
    Paths to include.
    """

    @classmethod
    def from_dict(cls, value: "FiltersDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FiltersDict":
        return _transform_to_json_value(self)  # type:ignore


class FiltersDict(TypedDict, total=False):
    """"""

    exclude: VariableOrList[str]
    """
    Paths to exclude.
    """

    include: VariableOrList[str]
    """
    Paths to include.
    """


FiltersParam = FiltersDict | Filters
