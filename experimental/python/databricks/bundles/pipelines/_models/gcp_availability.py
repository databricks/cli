from enum import Enum
from typing import Literal


class GcpAvailability(Enum):
    """
    This field determines whether the instance pool will contain preemptible
    VMs, on-demand VMs, or preemptible VMs with a fallback to on-demand VMs if the former is unavailable.
    """

    PREEMPTIBLE_GCP = "PREEMPTIBLE_GCP"
    ON_DEMAND_GCP = "ON_DEMAND_GCP"
    PREEMPTIBLE_WITH_FALLBACK_GCP = "PREEMPTIBLE_WITH_FALLBACK_GCP"


GcpAvailabilityParam = (
    Literal["PREEMPTIBLE_GCP", "ON_DEMAND_GCP", "PREEMPTIBLE_WITH_FALLBACK_GCP"]
    | GcpAvailability
)
