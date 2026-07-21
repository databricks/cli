from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ZendeskSupportOptions:
    """
    :meta private: [EXPERIMENTAL]

    Zendesk Support specific options for ingestion
    """

    start_date: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date in YYYY-MM-DD format for the initial sync.
    This determines the earliest date from which to sync historical data.
    """

    @classmethod
    def from_dict(cls, value: "ZendeskSupportOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ZendeskSupportOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class ZendeskSupportOptionsDict(TypedDict, total=False):
    """"""

    start_date: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date in YYYY-MM-DD format for the initial sync.
    This determines the earliest date from which to sync historical data.
    """


ZendeskSupportOptionsParam = ZendeskSupportOptionsDict | ZendeskSupportOptions
