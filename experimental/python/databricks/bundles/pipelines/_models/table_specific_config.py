from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.table_specific_config_scd_type import (
    TableSpecificConfigScdType,
    TableSpecificConfigScdTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TableSpecificConfig:
    """"""

    exclude_columns: VariableOrList[str] = field(default_factory=list)
    """
    A list of column names to be excluded for the ingestion.
    When not specified, include_columns fully controls what columns to be ingested.
    When specified, all other columns including future ones will be automatically included for ingestion.
    This field in mutually exclusive with `include_columns`.
    """

    include_columns: VariableOrList[str] = field(default_factory=list)
    """
    A list of column names to be included for the ingestion.
    When not specified, all columns except ones in exclude_columns will be included. Future
    columns will be automatically included.
    When specified, all other future columns will be automatically excluded from ingestion.
    This field in mutually exclusive with `exclude_columns`.
    """

    primary_keys: VariableOrList[str] = field(default_factory=list)
    """
    The primary key of the table used to apply changes.
    """

    salesforce_include_formula_fields: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
    """

    scd_type: VariableOrOptional[TableSpecificConfigScdType] = None
    """
    :meta private: [EXPERIMENTAL]
    
    The SCD type to use to ingest the table.
    """

    sequence_by: VariableOrList[str] = field(default_factory=list)
    """
    The column names specifying the logical order of events in the source data. Delta Live Tables uses this sequencing to handle change events that arrive out of order.
    """

    @classmethod
    def from_dict(cls, value: "TableSpecificConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TableSpecificConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class TableSpecificConfigDict(TypedDict, total=False):
    """"""

    exclude_columns: VariableOrList[str]
    """
    A list of column names to be excluded for the ingestion.
    When not specified, include_columns fully controls what columns to be ingested.
    When specified, all other columns including future ones will be automatically included for ingestion.
    This field in mutually exclusive with `include_columns`.
    """

    include_columns: VariableOrList[str]
    """
    A list of column names to be included for the ingestion.
    When not specified, all columns except ones in exclude_columns will be included. Future
    columns will be automatically included.
    When specified, all other future columns will be automatically excluded from ingestion.
    This field in mutually exclusive with `exclude_columns`.
    """

    primary_keys: VariableOrList[str]
    """
    The primary key of the table used to apply changes.
    """

    salesforce_include_formula_fields: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
    """

    scd_type: VariableOrOptional[TableSpecificConfigScdTypeParam]
    """
    :meta private: [EXPERIMENTAL]
    
    The SCD type to use to ingest the table.
    """

    sequence_by: VariableOrList[str]
    """
    The column names specifying the logical order of events in the source data. Delta Live Tables uses this sequencing to handle change events that arrive out of order.
    """


TableSpecificConfigParam = TableSpecificConfigDict | TableSpecificConfig
