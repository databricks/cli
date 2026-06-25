from enum import Enum
from typing import Literal


class AwsAvailability(Enum):
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.

    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    SPOT = "SPOT"
    ON_DEMAND = "ON_DEMAND"
    SPOT_WITH_FALLBACK = "SPOT_WITH_FALLBACK"


AwsAvailabilityParam = (
    Literal["SPOT", "ON_DEMAND", "SPOT_WITH_FALLBACK"] | AwsAvailability
)
