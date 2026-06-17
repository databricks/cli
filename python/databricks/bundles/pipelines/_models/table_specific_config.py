from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.auto_full_refresh_policy import (
    AutoFullRefreshPolicy,
    AutoFullRefreshPolicyParam,
)
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_table_specific_config_query_based_connector_config import (
    IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig,
    IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigParam,
)
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_workday_report_parameters import (
    IngestionPipelineDefinitionWorkdayReportParameters,
    IngestionPipelineDefinitionWorkdayReportParametersParam,
)
from databricks.bundles.pipelines._models.table_specific_config_scd_type import (
    TableSpecificConfigScdType,
    TableSpecificConfigScdTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TableSpecificConfig:
    """"""

    auto_full_refresh_policy: VariableOrOptional[AutoFullRefreshPolicy] = None
    """
    [Public Preview] (Optional, Mutable) Policy for auto full refresh, if enabled pipeline will automatically try
    to fix issues by doing a full refresh on the table in the retry run. auto_full_refresh_policy
    in table configuration will override the above level auto_full_refresh_policy.
    For example,
    {
    "auto_full_refresh_policy": {
    "enabled": true,
    "min_interval_hours": 23,
    }
    }
    If unspecified, auto full refresh is disabled.
    """

    exclude_columns: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] A list of column names to be excluded for the ingestion.
    When not specified, include_columns fully controls what columns to be ingested.
    When specified, all other columns including future ones will be automatically included for ingestion.
    This field in mutually exclusive with `include_columns`.
    """

    include_columns: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] A list of column names to be included for the ingestion.
    When not specified, all columns except ones in exclude_columns will be included. Future
    columns will be automatically included.
    When specified, all other future columns will be automatically excluded from ingestion.
    This field in mutually exclusive with `exclude_columns`.
    """

    primary_keys: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] The primary key of the table used to apply changes.
    """

    query_based_connector_config: VariableOrOptional[
        IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig
    ] = None
    """
    [Public Preview] Configurations that are only applicable for query-based ingestion connectors.
    """

    row_filter: VariableOrOptional[str] = None
    """
    [Public Preview] (Optional, Immutable) The row filter condition to be applied to the table.
    It must not contain the WHERE keyword, only the actual filter condition.
    It must be in DBSQL format.
    """

    salesforce_include_formula_fields: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
    """

    scd_type: VariableOrOptional[TableSpecificConfigScdType] = None
    """
    [Public Preview] The SCD type to use to ingest the table.
    """

    sequence_by: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] The column names specifying the logical order of events in the source data. Spark Declarative Pipelines uses this sequencing to handle change events that arrive out of order.
    """

    workday_report_parameters: VariableOrOptional[
        IngestionPipelineDefinitionWorkdayReportParameters
    ] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Additional custom parameters for Workday Report
    """

    @classmethod
    def from_dict(cls, value: "TableSpecificConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TableSpecificConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class TableSpecificConfigDict(TypedDict, total=False):
    """"""

    auto_full_refresh_policy: VariableOrOptional[AutoFullRefreshPolicyParam]
    """
    [Public Preview] (Optional, Mutable) Policy for auto full refresh, if enabled pipeline will automatically try
    to fix issues by doing a full refresh on the table in the retry run. auto_full_refresh_policy
    in table configuration will override the above level auto_full_refresh_policy.
    For example,
    {
    "auto_full_refresh_policy": {
    "enabled": true,
    "min_interval_hours": 23,
    }
    }
    If unspecified, auto full refresh is disabled.
    """

    exclude_columns: VariableOrList[str]
    """
    [Public Preview] A list of column names to be excluded for the ingestion.
    When not specified, include_columns fully controls what columns to be ingested.
    When specified, all other columns including future ones will be automatically included for ingestion.
    This field in mutually exclusive with `include_columns`.
    """

    include_columns: VariableOrList[str]
    """
    [Public Preview] A list of column names to be included for the ingestion.
    When not specified, all columns except ones in exclude_columns will be included. Future
    columns will be automatically included.
    When specified, all other future columns will be automatically excluded from ingestion.
    This field in mutually exclusive with `exclude_columns`.
    """

    primary_keys: VariableOrList[str]
    """
    [Public Preview] The primary key of the table used to apply changes.
    """

    query_based_connector_config: VariableOrOptional[
        IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigParam
    ]
    """
    [Public Preview] Configurations that are only applicable for query-based ingestion connectors.
    """

    row_filter: VariableOrOptional[str]
    """
    [Public Preview] (Optional, Immutable) The row filter condition to be applied to the table.
    It must not contain the WHERE keyword, only the actual filter condition.
    It must be in DBSQL format.
    """

    salesforce_include_formula_fields: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
    """

    scd_type: VariableOrOptional[TableSpecificConfigScdTypeParam]
    """
    [Public Preview] The SCD type to use to ingest the table.
    """

    sequence_by: VariableOrList[str]
    """
    [Public Preview] The column names specifying the logical order of events in the source data. Spark Declarative Pipelines uses this sequencing to handle change events that arrive out of order.
    """

    workday_report_parameters: VariableOrOptional[
        IngestionPipelineDefinitionWorkdayReportParametersParam
    ]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Additional custom parameters for Workday Report
    """


TableSpecificConfigParam = TableSpecificConfigDict | TableSpecificConfig
