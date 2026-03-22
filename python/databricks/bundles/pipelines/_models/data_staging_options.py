from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DataStagingOptions:
    """
    :meta private: [EXPERIMENTAL]

    Location of staged data storage
    """

    catalog_name: VariableOr[str]
    """
    (Required, Immutable) The name of the catalog for the connector's staging storage location.
    """

    schema_name: VariableOr[str]
    """
    (Required, Immutable) The name of the schema for the connector's staging storage location.
    """

    volume_name: VariableOrOptional[str] = None
    """
    (Optional) The Unity Catalog-compatible name for the storage location.
    This is the volume to use for the data that is extracted by the connector.
    Spark Declarative Pipelines system will automatically create the volume under the catalog and schema.
    For Combined Cdc Managed Ingestion pipelines default name for the volume would be :
    __databricks_ingestion_gateway_staging_data-$pipelineId
    """

    @classmethod
    def from_dict(cls, value: "DataStagingOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DataStagingOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class DataStagingOptionsDict(TypedDict, total=False):
    """"""

    catalog_name: VariableOr[str]
    """
    (Required, Immutable) The name of the catalog for the connector's staging storage location.
    """

    schema_name: VariableOr[str]
    """
    (Required, Immutable) The name of the schema for the connector's staging storage location.
    """

    volume_name: VariableOrOptional[str]
    """
    (Optional) The Unity Catalog-compatible name for the storage location.
    This is the volume to use for the data that is extracted by the connector.
    Spark Declarative Pipelines system will automatically create the volume under the catalog and schema.
    For Combined Cdc Managed Ingestion pipelines default name for the volume would be :
    __databricks_ingestion_gateway_staging_data-$pipelineId
    """


DataStagingOptionsParam = DataStagingOptionsDict | DataStagingOptions
