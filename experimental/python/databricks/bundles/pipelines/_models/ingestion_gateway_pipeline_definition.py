from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionGatewayPipelineDefinition:
    """
    :meta private: [EXPERIMENTAL]
    """

    connection_name: VariableOr[str]
    """
    Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
    """

    gateway_storage_catalog: VariableOr[str]
    """
    Required, Immutable. The name of the catalog for the gateway pipeline's storage location.
    """

    gateway_storage_schema: VariableOr[str]
    """
    Required, Immutable. The name of the schema for the gateway pipelines's storage location.
    """

    gateway_storage_name: VariableOrOptional[str] = None
    """
    Optional. The Unity Catalog-compatible name for the gateway storage location.
    This is the destination to use for the data that is extracted by the gateway.
    Delta Live Tables system will automatically create the storage location under the catalog and schema.
    """

    @classmethod
    def from_dict(cls, value: "IngestionGatewayPipelineDefinitionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "IngestionGatewayPipelineDefinitionDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionGatewayPipelineDefinitionDict(TypedDict, total=False):
    """"""

    connection_name: VariableOr[str]
    """
    Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
    """

    gateway_storage_catalog: VariableOr[str]
    """
    Required, Immutable. The name of the catalog for the gateway pipeline's storage location.
    """

    gateway_storage_schema: VariableOr[str]
    """
    Required, Immutable. The name of the schema for the gateway pipelines's storage location.
    """

    gateway_storage_name: VariableOrOptional[str]
    """
    Optional. The Unity Catalog-compatible name for the gateway storage location.
    This is the destination to use for the data that is extracted by the gateway.
    Delta Live Tables system will automatically create the storage location under the catalog and schema.
    """


IngestionGatewayPipelineDefinitionParam = (
    IngestionGatewayPipelineDefinitionDict | IngestionGatewayPipelineDefinition
)
