from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_workday_report_parameters_query_key_value import (
    IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue,
    IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinitionWorkdayReportParameters:
    """
    :meta private: [EXPERIMENTAL]
    """

    incremental: VariableOrOptional[bool] = None
    """
    [DEPRECATED] (Optional) Marks the report as incremental.
    This field is deprecated and should not be used. Use `parameters` instead. The incremental behavior is now
    controlled by the `parameters` field.
    """

    parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    Parameters for the Workday report. Each key represents the parameter name (e.g., "start_date", "end_date"),
    and the corresponding value is a SQL-like expression used to compute the parameter value at runtime.
    Example:
    {
    "start_date": "{ coalesce(current_offset(), date(\"2025-02-01\")) }",
    "end_date": "{ current_date() - INTERVAL 1 DAY }"
    }
    """

    report_parameters: VariableOrList[
        IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue
    ] = field(default_factory=list)
    """
    [DEPRECATED] (Optional) Additional custom parameters for Workday Report
    This field is deprecated and should not be used. Use `parameters` instead.
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

    incremental: VariableOrOptional[bool]
    """
    [DEPRECATED] (Optional) Marks the report as incremental.
    This field is deprecated and should not be used. Use `parameters` instead. The incremental behavior is now
    controlled by the `parameters` field.
    """

    parameters: VariableOrDict[str]
    """
    Parameters for the Workday report. Each key represents the parameter name (e.g., "start_date", "end_date"),
    and the corresponding value is a SQL-like expression used to compute the parameter value at runtime.
    Example:
    {
    "start_date": "{ coalesce(current_offset(), date(\"2025-02-01\")) }",
    "end_date": "{ current_date() - INTERVAL 1 DAY }"
    }
    """

    report_parameters: VariableOrList[
        IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueParam
    ]
    """
    [DEPRECATED] (Optional) Additional custom parameters for Workday Report
    This field is deprecated and should not be used. Use `parameters` instead.
    """


IngestionPipelineDefinitionWorkdayReportParametersParam = (
    IngestionPipelineDefinitionWorkdayReportParametersDict
    | IngestionPipelineDefinitionWorkdayReportParameters
)
