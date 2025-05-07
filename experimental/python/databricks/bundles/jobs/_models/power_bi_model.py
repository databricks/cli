from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.authentication_method import (
    AuthenticationMethod,
    AuthenticationMethodParam,
)
from databricks.bundles.jobs._models.storage_mode import StorageMode, StorageModeParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PowerBiModel:
    """"""

    authentication_method: VariableOrOptional[AuthenticationMethod] = None
    """
    How the published Power BI model authenticates to Databricks
    """

    model_name: VariableOrOptional[str] = None
    """
    The name of the Power BI model
    """

    overwrite_existing: VariableOrOptional[bool] = None
    """
    Whether to overwrite existing Power BI models
    """

    storage_mode: VariableOrOptional[StorageMode] = None
    """
    The default storage mode of the Power BI model
    """

    workspace_name: VariableOrOptional[str] = None
    """
    The name of the Power BI workspace of the model
    """

    @classmethod
    def from_dict(cls, value: "PowerBiModelDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PowerBiModelDict":
        return _transform_to_json_value(self)  # type:ignore


class PowerBiModelDict(TypedDict, total=False):
    """"""

    authentication_method: VariableOrOptional[AuthenticationMethodParam]
    """
    How the published Power BI model authenticates to Databricks
    """

    model_name: VariableOrOptional[str]
    """
    The name of the Power BI model
    """

    overwrite_existing: VariableOrOptional[bool]
    """
    Whether to overwrite existing Power BI models
    """

    storage_mode: VariableOrOptional[StorageModeParam]
    """
    The default storage mode of the Power BI model
    """

    workspace_name: VariableOrOptional[str]
    """
    The name of the Power BI workspace of the model
    """


PowerBiModelParam = PowerBiModelDict | PowerBiModel
