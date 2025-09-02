from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.ingestion_config import (
    IngestionConfig,
    IngestionConfigParam,
)
from databricks.bundles.pipelines._models.ingestion_source_type import (
    IngestionSourceType,
    IngestionSourceTypeParam,
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
    Immutable. The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with connectors for applications like Salesforce, Workday, and so on.
    """

    ingestion_gateway_id: VariableOrOptional[str] = None
    """
    Immutable. Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database. This is used with connectors to databases like SQL Server.
    """

    objects: VariableOrList[IngestionConfig] = field(default_factory=list)
    """
    Required. Settings specifying tables to replicate and the destination for the replicated tables.
    """

    source_type: VariableOrOptional[IngestionSourceType] = None
    """
    The type of the foreign source.
    The source type will be inferred from the source connection or ingestion gateway.
    This field is output only and will be ignored if provided.
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
    Immutable. The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with connectors for applications like Salesforce, Workday, and so on.
    """

    ingestion_gateway_id: VariableOrOptional[str]
    """
    Immutable. Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database. This is used with connectors to databases like SQL Server.
    """

    objects: VariableOrList[IngestionConfigParam]
    """
    Required. Settings specifying tables to replicate and the destination for the replicated tables.
    """

    source_type: VariableOrOptional[IngestionSourceTypeParam]
    """
    The type of the foreign source.
    The source type will be inferred from the source connection or ingestion gateway.
    This field is output only and will be ignored if provided.
    """

    table_configuration: VariableOrOptional[TableSpecificConfigParam]
    """
    Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
    """


IngestionPipelineDefinitionParam = (
    IngestionPipelineDefinitionDict | IngestionPipelineDefinition
)
