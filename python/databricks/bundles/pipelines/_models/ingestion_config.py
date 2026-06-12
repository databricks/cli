from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.report_spec import ReportSpec, ReportSpecParam
from databricks.bundles.pipelines._models.schema_spec import SchemaSpec, SchemaSpecParam
from databricks.bundles.pipelines._models.table_spec import TableSpec, TableSpecParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionConfig:
    """"""

    report: VariableOrOptional[ReportSpec] = None
    """
    Select a specific source report.
    """

    schema: VariableOrOptional[SchemaSpec] = None
    """
    Select all tables from a specific source schema.
    """

    table: VariableOrOptional[TableSpec] = None
    """
    Select a specific source table.
    """

    @classmethod
    def from_dict(cls, value: "IngestionConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "IngestionConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionConfigDict(TypedDict, total=False):
    """"""

    report: VariableOrOptional[ReportSpecParam]
    """
    Select a specific source report.
    """

    schema: VariableOrOptional[SchemaSpecParam]
    """
    Select all tables from a specific source schema.
    """

    table: VariableOrOptional[TableSpecParam]
    """
    Select a specific source table.
    """


IngestionConfigParam = IngestionConfigDict | IngestionConfig
