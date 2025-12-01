from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
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
    Required. Destination catalog to store tables.
    """

    destination_schema: VariableOr[str]
    """
    Required. Destination schema to store tables in. Tables with the same name as the source tables are created in this destination schema. The pipeline fails If a table with the same name already exists.
    """

    source_schema: VariableOr[str]
    """
    Required. Schema name in the source database.
    """

    source_catalog: VariableOrOptional[str] = None
    """
    The source catalog name. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfig] = None
    """
    Configuration settings to control the ingestion of tables. These settings are applied to all tables in this schema and override the table_configuration defined in the IngestionPipelineDefinition object.
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
    Required. Destination catalog to store tables.
    """

    destination_schema: VariableOr[str]
    """
    Required. Destination schema to store tables in. Tables with the same name as the source tables are created in this destination schema. The pipeline fails If a table with the same name already exists.
    """

    source_schema: VariableOr[str]
    """
    Required. Schema name in the source database.
    """

    source_catalog: VariableOrOptional[str]
    """
    The source catalog name. Might be optional depending on the type of source.
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    Configuration settings to control the ingestion of tables. These settings are applied to all tables in this schema and override the table_configuration defined in the IngestionPipelineDefinition object.
    """


SchemaSpecParam = SchemaSpecDict | SchemaSpec
