from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.table_specific_config import (
    TableSpecificConfig,
    TableSpecificConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ReportSpec:
    """"""

    destination_catalog: VariableOrOptional[str] = None
    """
    Required. Destination catalog to store table.
    """

    destination_schema: VariableOrOptional[str] = None
    """
    Required. Destination schema to store table.
    """

    destination_table: VariableOrOptional[str] = None
    """
    Required. Destination table name. The pipeline fails if a table with that name already exists.
    """

    source_url: VariableOrOptional[str] = None
    """
    Required. Report URL in the source system.
    """

    table_configuration: VariableOrOptional[TableSpecificConfig] = None
    """
    Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object.
    """

    @classmethod
    def from_dict(cls, value: "ReportSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ReportSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class ReportSpecDict(TypedDict, total=False):
    """"""

    destination_catalog: VariableOrOptional[str]
    """
    Required. Destination catalog to store table.
    """

    destination_schema: VariableOrOptional[str]
    """
    Required. Destination schema to store table.
    """

    destination_table: VariableOrOptional[str]
    """
    Required. Destination table name. The pipeline fails if a table with that name already exists.
    """

    source_url: VariableOrOptional[str]
    """
    Required. Report URL in the source system.
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object.
    """


ReportSpecParam = ReportSpecDict | ReportSpec
