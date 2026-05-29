from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.tik_tok_ads_options_tik_tok_data_level import (
    TikTokAdsOptionsTikTokDataLevel,
    TikTokAdsOptionsTikTokDataLevelParam,
)
from databricks.bundles.pipelines._models.tik_tok_ads_options_tik_tok_report_type import (
    TikTokAdsOptionsTikTokReportType,
    TikTokAdsOptionsTikTokReportTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TikTokAdsOptions:
    """
    :meta private: [EXPERIMENTAL]

    TikTok Ads specific options for ingestion
    """

    data_level: VariableOrOptional[TikTokAdsOptionsTikTokDataLevel] = None
    """
    (Optional) Data level for the report.
    If not specified, defaults to AUCTION_CAMPAIGN.
    """

    dimensions: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Dimensions to include in the report.
    Examples: "campaign_id", "adgroup_id", "ad_id", "stat_time_day", "stat_time_hour"
    If not specified, defaults to campaign_id.
    """

    lookback_window_days: VariableOrOptional[int] = None
    """
    (Optional) Number of days to look back for report tables during incremental sync
    to capture late-arriving conversions and attribution data.
    If not specified, defaults to 7 days.
    """

    metrics: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Metrics to include in the report.
    Examples: "spend", "impressions", "clicks", "conversion", "cpc"
    If not specified, defaults to basic metrics (spend, impressions, clicks, etc.)
    """

    query_lifetime: VariableOrOptional[bool] = None
    """
    (Optional) Whether to request lifetime metrics (all-time aggregated data).
    When true, the report returns all-time data.
    If not specified, defaults to false.
    """

    report_type: VariableOrOptional[TikTokAdsOptionsTikTokReportType] = None
    """
    (Optional) Report type for the TikTok Ads API.
    If not specified, defaults to BASIC.
    """

    sync_start_date: VariableOrOptional[str] = None
    """
    (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, defaults to 1 year of historical data for daily reports
    and 30 days for hourly reports.
    """

    @classmethod
    def from_dict(cls, value: "TikTokAdsOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TikTokAdsOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class TikTokAdsOptionsDict(TypedDict, total=False):
    """"""

    data_level: VariableOrOptional[TikTokAdsOptionsTikTokDataLevelParam]
    """
    (Optional) Data level for the report.
    If not specified, defaults to AUCTION_CAMPAIGN.
    """

    dimensions: VariableOrList[str]
    """
    (Optional) Dimensions to include in the report.
    Examples: "campaign_id", "adgroup_id", "ad_id", "stat_time_day", "stat_time_hour"
    If not specified, defaults to campaign_id.
    """

    lookback_window_days: VariableOrOptional[int]
    """
    (Optional) Number of days to look back for report tables during incremental sync
    to capture late-arriving conversions and attribution data.
    If not specified, defaults to 7 days.
    """

    metrics: VariableOrList[str]
    """
    (Optional) Metrics to include in the report.
    Examples: "spend", "impressions", "clicks", "conversion", "cpc"
    If not specified, defaults to basic metrics (spend, impressions, clicks, etc.)
    """

    query_lifetime: VariableOrOptional[bool]
    """
    (Optional) Whether to request lifetime metrics (all-time aggregated data).
    When true, the report returns all-time data.
    If not specified, defaults to false.
    """

    report_type: VariableOrOptional[TikTokAdsOptionsTikTokReportTypeParam]
    """
    (Optional) Report type for the TikTok Ads API.
    If not specified, defaults to BASIC.
    """

    sync_start_date: VariableOrOptional[str]
    """
    (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, defaults to 1 year of historical data for daily reports
    and 30 days for hourly reports.
    """


TikTokAdsOptionsParam = TikTokAdsOptionsDict | TikTokAdsOptions
