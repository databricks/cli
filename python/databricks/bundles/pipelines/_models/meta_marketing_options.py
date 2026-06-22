from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class MetaMarketingOptions:
    """
    Meta Marketing (Meta Ads) specific options for ingestion
    """

    action_attribution_windows: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_attribution_windows) Action attribution
    windows for insights reporting (e.g. "28d_click", "1d_view")
    """

    action_breakdowns: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_breakdowns) Action breakdowns
    """

    action_report_time: VariableOrOptional[str] = None
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_report_time) Timing used to report
    action statistics (impression, conversion, mixed, or lifetime)
    """

    breakdowns: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.breakdowns) Breakdowns to configure
    """

    custom_insights_lookback_window: VariableOrOptional[int] = None
    """
    [Beta] (Optional) Window in days to revisit data during sync to capture
    updated conversion data from the API, shared by prebuilt and custom reports.
    """

    level: VariableOrOptional[str] = None
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.level) Granularity of data to pull
    (account, ad, adset, campaign)
    """

    start_date: VariableOrOptional[str] = None
    """
    [Beta] (Optional) Start date in yyyy-MM-dd format (e.g. 2025-01-15). Data added
    after this date will be ingested, shared by prebuilt and custom reports.
    """

    time_increment: VariableOrOptional[str] = None
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.time_increment) Value in string by which to
    aggregate statistics (can take all_days, monthly or number of days)
    """

    @classmethod
    def from_dict(cls, value: "MetaMarketingOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "MetaMarketingOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class MetaMarketingOptionsDict(TypedDict, total=False):
    """"""

    action_attribution_windows: VariableOrList[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_attribution_windows) Action attribution
    windows for insights reporting (e.g. "28d_click", "1d_view")
    """

    action_breakdowns: VariableOrList[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_breakdowns) Action breakdowns
    """

    action_report_time: VariableOrOptional[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.action_report_time) Timing used to report
    action statistics (impression, conversion, mixed, or lifetime)
    """

    breakdowns: VariableOrList[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.breakdowns) Breakdowns to configure
    """

    custom_insights_lookback_window: VariableOrOptional[int]
    """
    [Beta] (Optional) Window in days to revisit data during sync to capture
    updated conversion data from the API, shared by prebuilt and custom reports.
    """

    level: VariableOrOptional[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.level) Granularity of data to pull
    (account, ad, adset, campaign)
    """

    start_date: VariableOrOptional[str]
    """
    [Beta] (Optional) Start date in yyyy-MM-dd format (e.g. 2025-01-15). Data added
    after this date will be ingested, shared by prebuilt and custom reports.
    """

    time_increment: VariableOrOptional[str]
    """
    [DEPRECATED] [Beta] (Optional, DEPRECATED — use custom_report_options.time_increment) Value in string by which to
    aggregate statistics (can take all_days, monthly or number of days)
    """


MetaMarketingOptionsParam = MetaMarketingOptionsDict | MetaMarketingOptions
