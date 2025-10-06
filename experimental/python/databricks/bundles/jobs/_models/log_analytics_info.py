from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class LogAnalyticsInfo:
    """"""

    log_analytics_primary_key: VariableOrOptional[str] = None
    """
    The primary key for the Azure Log Analytics agent configuration
    """

    log_analytics_workspace_id: VariableOrOptional[str] = None
    """
    The workspace ID for the Azure Log Analytics agent configuration
    """

    @classmethod
    def from_dict(cls, value: "LogAnalyticsInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "LogAnalyticsInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class LogAnalyticsInfoDict(TypedDict, total=False):
    """"""

    log_analytics_primary_key: VariableOrOptional[str]
    """
    The primary key for the Azure Log Analytics agent configuration
    """

    log_analytics_workspace_id: VariableOrOptional[str]
    """
    The workspace ID for the Azure Log Analytics agent configuration
    """


LogAnalyticsInfoParam = LogAnalyticsInfoDict | LogAnalyticsInfo
