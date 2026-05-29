from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

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
    If true, triggers a full refresh on the spark declarative pipeline.
    """

    full_refresh_selection: VariableOrList[str] = field(default_factory=list)
    """
    A list of tables to update with fullRefresh.
    """

    refresh_flow_selection: VariableOrList[str] = field(default_factory=list)
    """
    Flow names to selectively refresh. These are unioned with other selective refresh
    options (refresh_selection, full_refresh_selection) to determine the final set of flows to refresh.
    """

    refresh_selection: VariableOrList[str] = field(default_factory=list)
    """
    A list of tables to update without fullRefresh.
    """

    reset_checkpoint_selection: VariableOrList[str] = field(default_factory=list)
    """
    A list of streaming flows to reset checkpoints without clearing data.
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
    If true, triggers a full refresh on the spark declarative pipeline.
    """

    full_refresh_selection: VariableOrList[str]
    """
    A list of tables to update with fullRefresh.
    """

    refresh_flow_selection: VariableOrList[str]
    """
    Flow names to selectively refresh. These are unioned with other selective refresh
    options (refresh_selection, full_refresh_selection) to determine the final set of flows to refresh.
    """

    refresh_selection: VariableOrList[str]
    """
    A list of tables to update without fullRefresh.
    """

    reset_checkpoint_selection: VariableOrList[str]
    """
    A list of streaming flows to reset checkpoints without clearing data.
    """


PipelineTaskParam = PipelineTaskDict | PipelineTask
