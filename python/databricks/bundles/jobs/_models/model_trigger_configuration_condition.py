from enum import Enum
from typing import Literal


class ModelTriggerConfigurationCondition(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    MODEL_CREATED = "MODEL_CREATED"
    MODEL_VERSION_READY = "MODEL_VERSION_READY"
    MODEL_ALIAS_SET = "MODEL_ALIAS_SET"


ModelTriggerConfigurationConditionParam = (
    Literal["MODEL_CREATED", "MODEL_VERSION_READY", "MODEL_ALIAS_SET"]
    | ModelTriggerConfigurationCondition
)
