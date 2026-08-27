from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class MarketoOptions:
    """
    :meta private: [EXPERIMENTAL]

    Marketo specific options for ingestion
    """

    sync_start_date: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, complete history is ingested.
    """

    @classmethod
    def from_dict(cls, value: "MarketoOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "MarketoOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class MarketoOptionsDict(TypedDict, total=False):
    """"""

    sync_start_date: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, complete history is ingested.
    """


MarketoOptionsParam = MarketoOptionsDict | MarketoOptions
