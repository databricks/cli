from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ConfluenceConnectorOptions:
    """
    Confluence specific options for ingestion
    """

    include_confluence_spaces: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] (Optional) Spaces to filter Confluence data on
    """

    @classmethod
    def from_dict(cls, value: "ConfluenceConnectorOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ConfluenceConnectorOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class ConfluenceConnectorOptionsDict(TypedDict, total=False):
    """"""

    include_confluence_spaces: VariableOrList[str]
    """
    [Public Preview] (Optional) Spaces to filter Confluence data on
    """


ConfluenceConnectorOptionsParam = ConfluenceConnectorOptionsDict | ConfluenceConnectorOptions
