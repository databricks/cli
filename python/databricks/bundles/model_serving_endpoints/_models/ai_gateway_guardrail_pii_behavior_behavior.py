from enum import Enum
from typing import Literal


class AiGatewayGuardrailPiiBehaviorBehavior(Enum):
    NONE = "NONE"
    BLOCK = "BLOCK"
    MASK = "MASK"


AiGatewayGuardrailPiiBehaviorBehaviorParam = (
    Literal["NONE", "BLOCK", "MASK"] | AiGatewayGuardrailPiiBehaviorBehavior
)
