from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrDict

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinitionWorkdayReportParameters:
    """
    :meta private: [EXPERIMENTAL]
    """

    parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    Parameters for the Workday report. Each key represents the parameter name (e.g., "start_date", "end_date"),
    and the corresponding value is a SQL-like expression used to compute the parameter value at runtime.
    Example:
    {
    "start_date": "{ coalesce(current_offset(), date(\"2025-02-01\")) }",
    "end_date": "{ current_date() - INTERVAL 1 DAY }"
    }
    """

    @classmethod
    def from_dict(
        cls, value: "IngestionPipelineDefinitionWorkdayReportParametersDict"
    ) -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "IngestionPipelineDefinitionWorkdayReportParametersDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionPipelineDefinitionWorkdayReportParametersDict(TypedDict, total=False):
    """"""

    parameters: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Parameters for the Workday report. Each key represents the parameter name (e.g., "start_date", "end_date"),
    and the corresponding value is a SQL-like expression used to compute the parameter value at runtime.
    Example:
    {
    "start_date": "{ coalesce(current_offset(), date(\"2025-02-01\")) }",
    "end_date": "{ current_date() - INTERVAL 1 DAY }"
    }
    """


IngestionPipelineDefinitionWorkdayReportParametersParam = (
    IngestionPipelineDefinitionWorkdayReportParametersDict
    | IngestionPipelineDefinitionWorkdayReportParameters
)
