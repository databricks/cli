from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.power_bi_model import PowerBiModel, PowerBiModelParam
from databricks.bundles.jobs._models.power_bi_table import PowerBiTable, PowerBiTableParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PowerBiTask:
    """"""

    connection_resource_name: VariableOrOptional[str] = None
    """
    [Public Preview] The resource name of the UC connection to authenticate from Databricks to Power BI
    """

    power_bi_model: VariableOrOptional[PowerBiModel] = None
    """
    [Public Preview] The semantic model to update
    """

    refresh_after_update: VariableOrOptional[bool] = None
    """
    [Public Preview] Whether the model should be refreshed after the update
    """

    tables: VariableOrList[PowerBiTable] = field(default_factory=list)
    """
    [Public Preview] The tables to be exported to Power BI
    """

    warehouse_id: VariableOrOptional[str] = None
    """
    [Public Preview] The SQL warehouse ID to use as the Power BI data source
    """

    @classmethod
    def from_dict(cls, value: "PowerBiTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PowerBiTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class PowerBiTaskDict(TypedDict, total=False):
    """"""

    connection_resource_name: VariableOrOptional[str]
    """
    [Public Preview] The resource name of the UC connection to authenticate from Databricks to Power BI
    """

    power_bi_model: VariableOrOptional[PowerBiModelParam]
    """
    [Public Preview] The semantic model to update
    """

    refresh_after_update: VariableOrOptional[bool]
    """
    [Public Preview] Whether the model should be refreshed after the update
    """

    tables: VariableOrList[PowerBiTableParam]
    """
    [Public Preview] The tables to be exported to Power BI
    """

    warehouse_id: VariableOrOptional[str]
    """
    [Public Preview] The SQL warehouse ID to use as the Power BI data source
    """


PowerBiTaskParam = PowerBiTaskDict | PowerBiTask
