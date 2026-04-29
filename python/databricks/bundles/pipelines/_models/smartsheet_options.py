from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SmartsheetOptions:
    """
    :meta private: [EXPERIMENTAL]

    Smartsheet specific options for ingestion
    """

    enforce_schema: VariableOrOptional[bool] = None
    """
    (Optional) When true, maps each column to its Smartsheet-declared type (Text/Number/Date/
    Checkbox/etc.). Cells that do not conform to the declared type are set to NULL.
    When false, all columns land as STRING. Use false for sheets with irregular data or columns
    that frequently violate their own declared type.
    If not specified, defaults to true.
    """

    @classmethod
    def from_dict(cls, value: "SmartsheetOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SmartsheetOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class SmartsheetOptionsDict(TypedDict, total=False):
    """"""

    enforce_schema: VariableOrOptional[bool]
    """
    (Optional) When true, maps each column to its Smartsheet-declared type (Text/Number/Date/
    Checkbox/etc.). Cells that do not conform to the declared type are set to NULL.
    When false, all columns land as STRING. Use false for sheets with irregular data or columns
    that frequently violate their own declared type.
    If not specified, defaults to true.
    """


SmartsheetOptionsParam = SmartsheetOptionsDict | SmartsheetOptions
