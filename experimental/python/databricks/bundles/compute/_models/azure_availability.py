from enum import Enum
from typing import Literal


class AzureAvailability(Enum):
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    SPOT_AZURE = "SPOT_AZURE"
    ON_DEMAND_AZURE = "ON_DEMAND_AZURE"
    SPOT_WITH_FALLBACK_AZURE = "SPOT_WITH_FALLBACK_AZURE"


AzureAvailabilityParam = (
    Literal["SPOT_AZURE", "ON_DEMAND_AZURE", "SPOT_WITH_FALLBACK_AZURE"]
    | AzureAvailability
)
