from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AutoCaptureConfigInput:
    """"""

    catalog_name: VariableOrOptional[str] = None
    """
    The name of the catalog in Unity Catalog. NOTE: On update, you cannot change the catalog name if the inference table is already enabled.
    """

    enabled: VariableOrOptional[bool] = None
    """
    Indicates whether the inference table is enabled.
    """

    schema_name: VariableOrOptional[str] = None
    """
    The name of the schema in Unity Catalog. NOTE: On update, you cannot change the schema name if the inference table is already enabled.
    """

    table_name_prefix: VariableOrOptional[str] = None
    """
    The prefix of the table in Unity Catalog. NOTE: On update, you cannot change the prefix name if the inference table is already enabled.
    """

    @classmethod
    def from_dict(cls, value: "AutoCaptureConfigInputDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AutoCaptureConfigInputDict":
        return _transform_to_json_value(self)  # type:ignore


class AutoCaptureConfigInputDict(TypedDict, total=False):
    """"""

    catalog_name: VariableOrOptional[str]
    """
    The name of the catalog in Unity Catalog. NOTE: On update, you cannot change the catalog name if the inference table is already enabled.
    """

    enabled: VariableOrOptional[bool]
    """
    Indicates whether the inference table is enabled.
    """

    schema_name: VariableOrOptional[str]
    """
    The name of the schema in Unity Catalog. NOTE: On update, you cannot change the schema name if the inference table is already enabled.
    """

    table_name_prefix: VariableOrOptional[str]
    """
    The prefix of the table in Unity Catalog. NOTE: On update, you cannot change the prefix name if the inference table is already enabled.
    """


AutoCaptureConfigInputParam = AutoCaptureConfigInputDict | AutoCaptureConfigInput
