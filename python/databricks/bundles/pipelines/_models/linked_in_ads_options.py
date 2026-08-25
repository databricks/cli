from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.linked_in_ads_options_linked_in_ads_custom_report_options import (
    LinkedInAdsOptionsLinkedInAdsCustomReportOptions,
    LinkedInAdsOptionsLinkedInAdsCustomReportOptionsParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class LinkedInAdsOptions:
    """
    :meta private: [EXPERIMENTAL]

    LinkedIn Ads specific options for ingestion.
    sync_start_date and lookback_window_days apply to both the prebuilt analytics
    tables and custom reports. custom_report_options defines a custom (user-defined)
    adAnalytics report and is only valid on a table object.
    """

    custom_report_options: VariableOrOptional[
        LinkedInAdsOptionsLinkedInAdsCustomReportOptions
    ] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Custom report definition. Only valid on a table object. When set,
    the table is synthesized from /rest/adAnalytics using the finder, pivots,
    time granularity and metrics here. When unset, the table must match one of
    the connector's prebuilt sources.
    """

    lookback_window_days: VariableOrOptional[int] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Days to look back during incremental sync for late-arriving data.
    If not specified, defaults to 30 days.
    """

    sync_start_date: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync of report tables, YYYY-MM-DD.
    Earliest date from which to sync historical data; overrides the default
    when set. For finder attributedRevenueMetrics, this must be between 30 and
    366 days before today.
    If not specified, defaults to 1 year of history.
    """

    @classmethod
    def from_dict(cls, value: "LinkedInAdsOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "LinkedInAdsOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class LinkedInAdsOptionsDict(TypedDict, total=False):
    """"""

    custom_report_options: VariableOrOptional[
        LinkedInAdsOptionsLinkedInAdsCustomReportOptionsParam
    ]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Custom report definition. Only valid on a table object. When set,
    the table is synthesized from /rest/adAnalytics using the finder, pivots,
    time granularity and metrics here. When unset, the table must match one of
    the connector's prebuilt sources.
    """

    lookback_window_days: VariableOrOptional[int]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Days to look back during incremental sync for late-arriving data.
    If not specified, defaults to 30 days.
    """

    sync_start_date: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Start date for the initial sync of report tables, YYYY-MM-DD.
    Earliest date from which to sync historical data; overrides the default
    when set. For finder attributedRevenueMetrics, this must be between 30 and
    366 days before today.
    If not specified, defaults to 1 year of history.
    """


LinkedInAdsOptionsParam = LinkedInAdsOptionsDict | LinkedInAdsOptions
