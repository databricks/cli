from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrDict

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ApiSourceConnectorConfig:
    """
    :meta private: [EXPERIMENTAL]

    Top-level configuration for API Source connectors with arbitrary configuration.
    """

    configs: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Arbitrary key-value configuration values for the API Source connector.
    """

    @classmethod
    def from_dict(cls, value: "ApiSourceConnectorConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ApiSourceConnectorConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class ApiSourceConnectorConfigDict(TypedDict, total=False):
    """"""

    configs: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Arbitrary key-value configuration values for the API Source connector.
    """


ApiSourceConnectorConfigParam = ApiSourceConnectorConfigDict | ApiSourceConnectorConfig
