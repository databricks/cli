from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.pipelines._models.connector_options import (
    ConnectorOptions,
    ConnectorOptionsParam,
)
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_fanout_options import (
    IngestionPipelineDefinitionFanoutOptions,
    IngestionPipelineDefinitionFanoutOptionsParam,
)
from databricks.bundles.pipelines._models.table_specific_config import (
    TableSpecificConfig,
    TableSpecificConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SchemaSpec:
    """"""

    destination_catalog: VariableOr[str]
    """
    [Public Preview] Required. Destination catalog to store tables.
    """

    destination_schema: VariableOr[str]
    """
    [Public Preview] Required. Destination schema to store tables in. Tables with the same name as the source tables are created in this destination schema. The pipeline fails If a table with the same name already exists.
    """

    source_schema: VariableOr[str]
    """
    [Public Preview] Schema name in the source database. Currently required; this field will become optional in
    an upcoming release, since some source types (for example streaming / message-bus connectors)
    do not use it. When that change ships, this field's type in the generated SDKs and CLI will
    change from required to optional (nullable); clients that assume it is always present should
    handle its absence.
    """

    connector_options: VariableOrOptional[ConnectorOptions] = None
    """
    [Public Preview] (Optional) Source Specific Connector Options
    """

    fanout_options: VariableOrOptional[IngestionPipelineDefinitionFanoutOptions] = None
    """
    [Beta] Fanout options for multi-table routing from streaming sources.
    When set, records are routed to destination tables based on a
    per-record routing key. The key value becomes the table name:
    {destination_catalog}.{destination_schema}.{key_value}.
    """

    source_catalog: VariableOrOptional[str] = None
    """
    [Public Preview] The source catalog name. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfig] = None
    """
    [Public Preview] Configuration settings to control the ingestion of tables. These settings are applied to all tables in this schema and override the table_configuration defined in the IngestionPipelineDefinition object.
    """

    @classmethod
    def from_dict(cls, value: "SchemaSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SchemaSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class SchemaSpecDict(TypedDict, total=False):
    """"""

    destination_catalog: VariableOr[str]
    """
    [Public Preview] Required. Destination catalog to store tables.
    """

    destination_schema: VariableOr[str]
    """
    [Public Preview] Required. Destination schema to store tables in. Tables with the same name as the source tables are created in this destination schema. The pipeline fails If a table with the same name already exists.
    """

    source_schema: VariableOr[str]
    """
    [Public Preview] Schema name in the source database. Currently required; this field will become optional in
    an upcoming release, since some source types (for example streaming / message-bus connectors)
    do not use it. When that change ships, this field's type in the generated SDKs and CLI will
    change from required to optional (nullable); clients that assume it is always present should
    handle its absence.
    """

    connector_options: VariableOrOptional[ConnectorOptionsParam]
    """
    [Public Preview] (Optional) Source Specific Connector Options
    """

    fanout_options: VariableOrOptional[IngestionPipelineDefinitionFanoutOptionsParam]
    """
    [Beta] Fanout options for multi-table routing from streaming sources.
    When set, records are routed to destination tables based on a
    per-record routing key. The key value becomes the table name:
    {destination_catalog}.{destination_schema}.{key_value}.
    """

    source_catalog: VariableOrOptional[str]
    """
    [Public Preview] The source catalog name. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    [Public Preview] Configuration settings to control the ingestion of tables. These settings are applied to all tables in this schema and override the table_configuration defined in the IngestionPipelineDefinition object.
    """


SchemaSpecParam = SchemaSpecDict | SchemaSpec
