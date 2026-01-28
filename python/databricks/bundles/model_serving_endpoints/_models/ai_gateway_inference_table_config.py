from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiGatewayInferenceTableConfig:
    """"""

    catalog_name: VariableOrOptional[str] = None
    """
    The name of the catalog in Unity Catalog. Required when enabling inference tables.
    NOTE: On update, you have to disable inference table first in order to change the catalog name.
    """

    enabled: VariableOrOptional[bool] = None
    """
    Indicates whether the inference table is enabled.
    """

    schema_name: VariableOrOptional[str] = None
    """
    The name of the schema in Unity Catalog. Required when enabling inference tables.
    NOTE: On update, you have to disable inference table first in order to change the schema name.
    """

    table_name_prefix: VariableOrOptional[str] = None
    """
    The prefix of the table in Unity Catalog.
    NOTE: On update, you have to disable inference table first in order to change the prefix name.
    """

    @classmethod
    def from_dict(cls, value: "AiGatewayInferenceTableConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiGatewayInferenceTableConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class AiGatewayInferenceTableConfigDict(TypedDict, total=False):
    """"""

    catalog_name: VariableOrOptional[str]
    """
    The name of the catalog in Unity Catalog. Required when enabling inference tables.
    NOTE: On update, you have to disable inference table first in order to change the catalog name.
    """

    enabled: VariableOrOptional[bool]
    """
    Indicates whether the inference table is enabled.
    """

    schema_name: VariableOrOptional[str]
    """
    The name of the schema in Unity Catalog. Required when enabling inference tables.
    NOTE: On update, you have to disable inference table first in order to change the schema name.
    """

    table_name_prefix: VariableOrOptional[str]
    """
    The prefix of the table in Unity Catalog.
    NOTE: On update, you have to disable inference table first in order to change the prefix name.
    """


AiGatewayInferenceTableConfigParam = (
    AiGatewayInferenceTableConfigDict | AiGatewayInferenceTableConfig
)
