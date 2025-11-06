from enum import Enum
from typing import Literal


class EbsVolumeType(Enum):
    """
    All EBS volume types that Databricks supports.
    See https://aws.amazon.com/ebs/details/ for details.
    """

    GENERAL_PURPOSE_SSD = "GENERAL_PURPOSE_SSD"
    THROUGHPUT_OPTIMIZED_HDD = "THROUGHPUT_OPTIMIZED_HDD"


EbsVolumeTypeParam = (
    Literal["GENERAL_PURPOSE_SSD", "THROUGHPUT_OPTIMIZED_HDD"] | EbsVolumeType
)
