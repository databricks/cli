from enum import Enum
from typing import Literal


class HardwareAcceleratorType(Enum):
    """
    HardwareAcceleratorType: The type of hardware accelerator to use for compute workloads.
    NOTE: This enum is referenced and is intended to be used by other Databricks services
    that need to specify hardware accelerator requirements for AI compute workloads.
    """

    GPU_1X_A10 = "GPU_1xA10"
    GPU_8X_H100 = "GPU_8xH100"


HardwareAcceleratorTypeParam = (
    Literal["GPU_1xA10", "GPU_8xH100"] | HardwareAcceleratorType
)
