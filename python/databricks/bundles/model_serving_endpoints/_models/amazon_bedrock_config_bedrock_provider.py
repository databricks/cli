from enum import Enum
from typing import Literal


class AmazonBedrockConfigBedrockProvider(Enum):
    ANTHROPIC = "anthropic"
    COHERE = "cohere"
    AI21LABS = "ai21labs"
    AMAZON = "amazon"


AmazonBedrockConfigBedrockProviderParam = (
    Literal["anthropic", "cohere", "ai21labs", "amazon"]
    | AmazonBedrockConfigBedrockProvider
)
