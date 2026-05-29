from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JiraConnectorOptions:
    """
    Jira specific options for ingestion
    """

    include_jira_spaces: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Projects to filter Jira data on
    """

    @classmethod
    def from_dict(cls, value: "JiraConnectorOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JiraConnectorOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class JiraConnectorOptionsDict(TypedDict, total=False):
    """"""

    include_jira_spaces: VariableOrList[str]
    """
    (Optional) Projects to filter Jira data on
    """


JiraConnectorOptionsParam = JiraConnectorOptionsDict | JiraConnectorOptions
