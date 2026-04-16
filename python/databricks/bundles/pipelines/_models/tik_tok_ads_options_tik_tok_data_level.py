from enum import Enum
from typing import Literal


class TikTokAdsOptionsTikTokDataLevel(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Data level for TikTok Ads report aggregation.
    """

    AUCTION_ADVERTISER = "AUCTION_ADVERTISER"
    AUCTION_CAMPAIGN = "AUCTION_CAMPAIGN"
    AUCTION_ADGROUP = "AUCTION_ADGROUP"
    AUCTION_AD = "AUCTION_AD"


TikTokAdsOptionsTikTokDataLevelParam = (
    Literal["AUCTION_ADVERTISER", "AUCTION_CAMPAIGN", "AUCTION_ADGROUP", "AUCTION_AD"]
    | TikTokAdsOptionsTikTokDataLevel
)
