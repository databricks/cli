from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrDict

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ApiSourceConnectorOptions:
    """
    :meta private: [EXPERIMENTAL]

    Options for API Source connectors with arbitrary configuration.
    """

    options: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Arbitrary key-value configuration options for the API Source connector.
    """

    @classmethod
    def from_dict(cls, value: "ApiSourceConnectorOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ApiSourceConnectorOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class ApiSourceConnectorOptionsDict(TypedDict, total=False):
    """"""

    options: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Arbitrary key-value configuration options for the API Source connector.
    """


ApiSourceConnectorOptionsParam = (
    ApiSourceConnectorOptionsDict | ApiSourceConnectorOptions
)
