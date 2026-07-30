from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.pipelines._models.connector_options import (
    ConnectorOptions,
    ConnectorOptionsParam,
)
from databricks.bundles.pipelines._models.table_specific_config import (
    TableSpecificConfig,
    TableSpecificConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TableSpec:
    """"""

    destination_catalog: VariableOr[str]
    """
    [Public Preview] Required. Destination catalog to store table.
    """

    destination_schema: VariableOr[str]
    """
    [Public Preview] Required. Destination schema to store table.
    """

    source_table: VariableOr[str]
    """
    [Public Preview] Required. Table name in the source database.
    """

    connector_options: VariableOrOptional[ConnectorOptions] = None
    """
    [Public Preview] (Optional) Source Specific Connector Options
    """

    destination_table: VariableOrOptional[str] = None
    """
    [Public Preview] Optional. Destination table name. The pipeline fails if a table with that name already exists. If not set, the source table name is used.
    """

    source_catalog: VariableOrOptional[str] = None
    """
    [Public Preview] Source catalog name. Might be optional depending on the type of source.
    """

    source_schema: VariableOrOptional[str] = None
    """
    [Public Preview] Schema name in the source database. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfig] = None
    """
    [Public Preview] Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object and the SchemaSpec.
    """

    @classmethod
    def from_dict(cls, value: "TableSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TableSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class TableSpecDict(TypedDict, total=False):
    """"""

    destination_catalog: VariableOr[str]
    """
    [Public Preview] Required. Destination catalog to store table.
    """

    destination_schema: VariableOr[str]
    """
    [Public Preview] Required. Destination schema to store table.
    """

    source_table: VariableOr[str]
    """
    [Public Preview] Required. Table name in the source database.
    """

    connector_options: VariableOrOptional[ConnectorOptionsParam]
    """
    [Public Preview] (Optional) Source Specific Connector Options
    """

    destination_table: VariableOrOptional[str]
    """
    [Public Preview] Optional. Destination table name. The pipeline fails if a table with that name already exists. If not set, the source table name is used.
    """

    source_catalog: VariableOrOptional[str]
    """
    [Public Preview] Source catalog name. Might be optional depending on the type of source.
    """

    source_schema: VariableOrOptional[str]
    """
    [Public Preview] Schema name in the source database. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    [Public Preview] Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object and the SchemaSpec.
    """


TableSpecParam = TableSpecDict | TableSpec
