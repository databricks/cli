from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.ingestion_config import (
    IngestionConfig,
    IngestionConfigParam,
)
from databricks.bundles.pipelines._models.operation_time_window import (
    OperationTimeWindow,
    OperationTimeWindowParam,
)
from databricks.bundles.pipelines._models.source_config import (
    SourceConfig,
    SourceConfigParam,
)
from databricks.bundles.pipelines._models.table_specific_config import (
    TableSpecificConfig,
    TableSpecificConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinition:
    """"""

    connection_name: VariableOrOptional[str] = None
    """
    The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with
    both connectors for applications like Salesforce, Workday, and so on, and also database connectors like Oracle,
    (connector_type = QUERY_BASED OR connector_type = CDC).
    If connection name corresponds to database connectors like Oracle, and connector_type is not provided then
    connector_type defaults to QUERY_BASED. If connector_type is passed as CDC we use Combined Cdc Managed Ingestion
    pipeline.
    Under certain conditions, this can be replaced with ingestion_gateway_id to change the connector to Cdc Managed
    Ingestion Pipeline with Gateway pipeline.
    """

    full_refresh_window: VariableOrOptional[OperationTimeWindow] = None
    """
    (Optional) A window that specifies a set of time ranges for snapshot queries in CDC.
    """

    ingest_from_uc_foreign_catalog: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Immutable. If set to true, the pipeline will ingest tables from the
    UC foreign catalogs directly without the need to specify a UC connection or ingestion gateway.
    The `source_catalog` fields in objects of IngestionConfig are interpreted as
    the UC foreign catalogs to ingest from.
    """

    ingestion_gateway_id: VariableOrOptional[str] = None
    """
    Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database.
    This is used with CDC connectors to databases like SQL Server using a gateway pipeline (connector_type = CDC).
    Under certain conditions, this can be replaced with connection_name to change the connector to Combined Cdc
    Managed Ingestion Pipeline.
    """

    netsuite_jar_path: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    objects: VariableOrList[IngestionConfig] = field(default_factory=list)
    """
    Required. Settings specifying tables to replicate and the destination for the replicated tables.
    """

    source_configurations: VariableOrList[SourceConfig] = field(default_factory=list)
    """
    Top-level source configurations
    """

    table_configuration: VariableOrOptional[TableSpecificConfig] = None
    """
    Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
    """

    @classmethod
    def from_dict(cls, value: "IngestionPipelineDefinitionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "IngestionPipelineDefinitionDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionPipelineDefinitionDict(TypedDict, total=False):
    """"""

    connection_name: VariableOrOptional[str]
    """
    The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with
    both connectors for applications like Salesforce, Workday, and so on, and also database connectors like Oracle,
    (connector_type = QUERY_BASED OR connector_type = CDC).
    If connection name corresponds to database connectors like Oracle, and connector_type is not provided then
    connector_type defaults to QUERY_BASED. If connector_type is passed as CDC we use Combined Cdc Managed Ingestion
    pipeline.
    Under certain conditions, this can be replaced with ingestion_gateway_id to change the connector to Cdc Managed
    Ingestion Pipeline with Gateway pipeline.
    """

    full_refresh_window: VariableOrOptional[OperationTimeWindowParam]
    """
    (Optional) A window that specifies a set of time ranges for snapshot queries in CDC.
    """

    ingest_from_uc_foreign_catalog: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    Immutable. If set to true, the pipeline will ingest tables from the
    UC foreign catalogs directly without the need to specify a UC connection or ingestion gateway.
    The `source_catalog` fields in objects of IngestionConfig are interpreted as
    the UC foreign catalogs to ingest from.
    """

    ingestion_gateway_id: VariableOrOptional[str]
    """
    Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database.
    This is used with CDC connectors to databases like SQL Server using a gateway pipeline (connector_type = CDC).
    Under certain conditions, this can be replaced with connection_name to change the connector to Combined Cdc
    Managed Ingestion Pipeline.
    """

    netsuite_jar_path: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    """

    objects: VariableOrList[IngestionConfigParam]
    """
    Required. Settings specifying tables to replicate and the destination for the replicated tables.
    """

    source_configurations: VariableOrList[SourceConfigParam]
    """
    Top-level source configurations
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
    """


IngestionPipelineDefinitionParam = (
    IngestionPipelineDefinitionDict | IngestionPipelineDefinition
)
