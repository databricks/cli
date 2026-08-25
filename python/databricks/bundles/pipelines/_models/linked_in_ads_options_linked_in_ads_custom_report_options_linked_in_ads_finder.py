from enum import Enum
from typing import Literal


class LinkedInAdsOptionsLinkedInAdsCustomReportOptionsLinkedInAdsFinder(Enum):
    """
    :meta private: [EXPERIMENTAL]

    adAnalytics finder. Determines call shape, valid pivots, and metric
    requirements.
    """

    ANALYTICS = "ANALYTICS"
    STATISTICS = "STATISTICS"
    ATTRIBUTED_REVENUE_METRICS = "ATTRIBUTED_REVENUE_METRICS"


LinkedInAdsOptionsLinkedInAdsCustomReportOptionsLinkedInAdsFinderParam = (
    Literal["ANALYTICS", "STATISTICS", "ATTRIBUTED_REVENUE_METRICS"]
    | LinkedInAdsOptionsLinkedInAdsCustomReportOptionsLinkedInAdsFinder
)
