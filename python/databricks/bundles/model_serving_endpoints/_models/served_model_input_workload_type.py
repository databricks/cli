from enum import Enum
from typing import Literal


class ServedModelInputWorkloadType(Enum):
    """
    Please keep this in sync with with workload types in InferenceEndpointEntities.scala
    """

    CPU = "CPU"
    GPU_MEDIUM = "GPU_MEDIUM"
    GPU_SMALL = "GPU_SMALL"
    GPU_LARGE = "GPU_LARGE"
    MULTIGPU_MEDIUM = "MULTIGPU_MEDIUM"


ServedModelInputWorkloadTypeParam = (
    Literal["CPU", "GPU_MEDIUM", "GPU_SMALL", "GPU_LARGE", "MULTIGPU_MEDIUM"]
    | ServedModelInputWorkloadType
)
