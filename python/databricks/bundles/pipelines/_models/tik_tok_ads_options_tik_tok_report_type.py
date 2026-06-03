from enum import Enum
from typing import Literal


class TikTokAdsOptionsTikTokReportType(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Report type for TikTok Ads API.
    """

    BASIC = "BASIC"
    AUDIENCE = "AUDIENCE"
    PLAYABLE_AD = "PLAYABLE_AD"
    DSA = "DSA"
    BUSINESS_CENTER = "BUSINESS_CENTER"
    GMV_MAX = "GMV_MAX"


TikTokAdsOptionsTikTokReportTypeParam = (
    Literal["BASIC", "AUDIENCE", "PLAYABLE_AD", "DSA", "BUSINESS_CENTER", "GMV_MAX"]
    | TikTokAdsOptionsTikTokReportType
)
