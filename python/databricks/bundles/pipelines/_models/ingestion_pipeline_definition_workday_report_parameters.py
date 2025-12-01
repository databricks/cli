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
    """"""

    incremental: VariableOrOptional[bool] = None

    parameters: VariableOrDict[str] = field(default_factory=dict)

    report_parameters: VariableOrList[
        IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue
    ] = field(default_factory=list)

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

    parameters: VariableOrDict[str]

    report_parameters: VariableOrList[
        IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueParam
    ]


IngestionPipelineDefinitionWorkdayReportParametersParam = (
    IngestionPipelineDefinitionWorkdayReportParametersDict
    | IngestionPipelineDefinitionWorkdayReportParameters
)
