from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TikTokAdsOptions:
    """
    :meta private: [EXPERIMENTAL]

    TikTok Ads specific options for ingestion
    """

    lookback_window_days: VariableOrOptional[int] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Number of days to look back for report tables during incremental sync
    to capture late-arriving conversions and attribution data.
    """

    sync_start_date: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    """

    @classmethod
    def from_dict(cls, value: "TikTokAdsOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TikTokAdsOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class TikTokAdsOptionsDict(TypedDict, total=False):
    """"""

    lookback_window_days: VariableOrOptional[int]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Number of days to look back for report tables during incremental sync
    to capture late-arriving conversions and attribution data.
    """

    sync_start_date: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    """


TikTokAdsOptionsParam = TikTokAdsOptionsDict | TikTokAdsOptions
