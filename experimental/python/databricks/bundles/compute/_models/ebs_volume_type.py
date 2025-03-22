from enum import Enum
from typing import Literal


class EbsVolumeType(Enum):
    """
    The type of EBS volumes that will be launched with this cluster.
    """

    GENERAL_PURPOSE_SSD = "GENERAL_PURPOSE_SSD"
    THROUGHPUT_OPTIMIZED_HDD = "THROUGHPUT_OPTIMIZED_HDD"


EbsVolumeTypeParam = (
    Literal["GENERAL_PURPOSE_SSD", "THROUGHPUT_OPTIMIZED_HDD"] | EbsVolumeType
)
