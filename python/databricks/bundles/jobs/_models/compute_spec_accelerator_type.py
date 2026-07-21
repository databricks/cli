from enum import Enum
from typing import Literal


class ComputeSpecAcceleratorType(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Customer-facing AcceleratorType: hardware accelerator type for the
    AiRuntime workload. Per-node accelerator count is encoded in the value
    name (e.g. `GPU_8xH100` means 8 H100s per node).
    """

    GPU_1X_A10 = "GPU_1xA10"
    GPU_1X_H100 = "GPU_1xH100"
    GPU_8X_H100 = "GPU_8xH100"


ComputeSpecAcceleratorTypeParam = (
    Literal["GPU_1xA10", "GPU_1xH100", "GPU_8xH100"] | ComputeSpecAcceleratorType
)
