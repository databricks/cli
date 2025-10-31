from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.storage_mode import StorageMode, StorageModeParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PowerBiTable:
    """"""

    catalog: VariableOrOptional[str] = None
    """
    The catalog name in Databricks
    """

    name: VariableOrOptional[str] = None
    """
    The table name in Databricks
    """

    schema: VariableOrOptional[str] = None
    """
    The schema name in Databricks
    """

    storage_mode: VariableOrOptional[StorageMode] = None
    """
    The Power BI storage mode of the table
    """

    @classmethod
    def from_dict(cls, value: "PowerBiTableDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PowerBiTableDict":
        return _transform_to_json_value(self)  # type:ignore


class PowerBiTableDict(TypedDict, total=False):
    """"""

    catalog: VariableOrOptional[str]
    """
    The catalog name in Databricks
    """

    name: VariableOrOptional[str]
    """
    The table name in Databricks
    """

    schema: VariableOrOptional[str]
    """
    The schema name in Databricks
    """

    storage_mode: VariableOrOptional[StorageModeParam]
    """
    The Power BI storage mode of the table
    """


PowerBiTableParam = PowerBiTableDict | PowerBiTable
