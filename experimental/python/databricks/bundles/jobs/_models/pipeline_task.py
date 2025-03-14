from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineTask:
    """"""

    pipeline_id: VariableOr[str]
    """
    The full name of the pipeline task to execute.
    """

    full_refresh: VariableOrOptional[bool] = None
    """
    If true, triggers a full refresh on the delta live table.
    """

    @classmethod
    def from_dict(cls, value: "PipelineTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineTaskDict(TypedDict, total=False):
    """"""

    pipeline_id: VariableOr[str]
    """
    The full name of the pipeline task to execute.
    """

    full_refresh: VariableOrOptional[bool]
    """
    If true, triggers a full refresh on the delta live table.
    """


PipelineTaskParam = PipelineTaskDict | PipelineTask
