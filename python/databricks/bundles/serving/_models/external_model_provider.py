from enum import Enum
from typing import Literal


class ExternalModelProvider(Enum):
    AI21LABS = "ai21labs"
    ANTHROPIC = "anthropic"
    AMAZON_BEDROCK = "amazon-bedrock"
    COHERE = "cohere"
    DATABRICKS_MODEL_SERVING = "databricks-model-serving"
    GOOGLE_CLOUD_VERTEX_AI = "google-cloud-vertex-ai"
    OPENAI = "openai"
    PALM = "palm"
    CUSTOM = "custom"


ExternalModelProviderParam = (
    Literal[
        "ai21labs",
        "anthropic",
        "amazon-bedrock",
        "cohere",
        "databricks-model-serving",
        "google-cloud-vertex-ai",
        "openai",
        "palm",
        "custom",
    ]
    | ExternalModelProvider
)
