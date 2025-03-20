from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineParams:
    """"""

    full_refresh: VariableOrOptional[bool] = None
    """
    If true, triggers a full refresh on the delta live table.
    """

    @classmethod
    def from_dict(cls, value: "PipelineParamsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineParamsDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineParamsDict(TypedDict, total=False):
    """"""

    full_refresh: VariableOrOptional[bool]
    """
    If true, triggers a full refresh on the delta live table.
    """


PipelineParamsParam = PipelineParamsDict | PipelineParams
