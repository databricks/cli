from enum import Enum
from typing import Literal


class EnablePredictiveOptimization(Enum):
    DISABLE = "DISABLE"
    ENABLE = "ENABLE"
    INHERIT = "INHERIT"


EnablePredictiveOptimizationParam = (
    Literal["DISABLE", "ENABLE", "INHERIT"] | EnablePredictiveOptimization
)
