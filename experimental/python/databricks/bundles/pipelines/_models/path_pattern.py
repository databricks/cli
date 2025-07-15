from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PathPattern:
    """"""

    include: VariableOrOptional[str] = None
    """
    The source code to include for pipelines
    """

    @classmethod
    def from_dict(cls, value: "PathPatternDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PathPatternDict":
        return _transform_to_json_value(self)  # type:ignore


class PathPatternDict(TypedDict, total=False):
    """"""

    include: VariableOrOptional[str]
    """
    The source code to include for pipelines
    """


PathPatternParam = PathPatternDict | PathPattern
