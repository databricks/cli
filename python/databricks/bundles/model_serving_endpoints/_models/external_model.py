from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.ai21_labs_config import (
    Ai21LabsConfig,
    Ai21LabsConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.amazon_bedrock_config import (
    AmazonBedrockConfig,
    AmazonBedrockConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.anthropic_config import (
    AnthropicConfig,
    AnthropicConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.cohere_config import (
    CohereConfig,
    CohereConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.custom_provider_config import (
    CustomProviderConfig,
    CustomProviderConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.databricks_model_serving_config import (
    DatabricksModelServingConfig,
    DatabricksModelServingConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.external_model_provider import (
    ExternalModelProvider,
    ExternalModelProviderParam,
)
from databricks.bundles.model_serving_endpoints._models.google_cloud_vertex_ai_config import (
    GoogleCloudVertexAiConfig,
    GoogleCloudVertexAiConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.open_ai_config import (
    OpenAiConfig,
    OpenAiConfigParam,
)
from databricks.bundles.model_serving_endpoints._models.pa_lm_config import (
    PaLmConfig,
    PaLmConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ExternalModel:
    """"""

    name: VariableOr[str]
    """
    The name of the external model.
    """

    provider: VariableOr[ExternalModelProvider]
    """
    The name of the provider for the external model. Currently, the supported providers are 'ai21labs', 'anthropic', 'amazon-bedrock', 'cohere', 'databricks-model-serving', 'google-cloud-vertex-ai', 'openai', 'palm', and 'custom'.
    """

    task: VariableOr[str]
    """
    The task type of the external model.
    """

    ai21labs_config: VariableOrOptional[Ai21LabsConfig] = None
    """
    AI21Labs Config. Only required if the provider is 'ai21labs'.
    """

    amazon_bedrock_config: VariableOrOptional[AmazonBedrockConfig] = None
    """
    Amazon Bedrock Config. Only required if the provider is 'amazon-bedrock'.
    """

    anthropic_config: VariableOrOptional[AnthropicConfig] = None
    """
    Anthropic Config. Only required if the provider is 'anthropic'.
    """

    cohere_config: VariableOrOptional[CohereConfig] = None
    """
    Cohere Config. Only required if the provider is 'cohere'.
    """

    custom_provider_config: VariableOrOptional[CustomProviderConfig] = None
    """
    Custom Provider Config. Only required if the provider is 'custom'.
    """

    databricks_model_serving_config: VariableOrOptional[
        DatabricksModelServingConfig
    ] = None
    """
    Databricks Model Serving Config. Only required if the provider is 'databricks-model-serving'.
    """

    google_cloud_vertex_ai_config: VariableOrOptional[GoogleCloudVertexAiConfig] = None
    """
    Google Cloud Vertex AI Config. Only required if the provider is 'google-cloud-vertex-ai'.
    """

    openai_config: VariableOrOptional[OpenAiConfig] = None
    """
    OpenAI Config. Only required if the provider is 'openai'.
    """

    palm_config: VariableOrOptional[PaLmConfig] = None
    """
    PaLM Config. Only required if the provider is 'palm'.
    """

    @classmethod
    def from_dict(cls, value: "ExternalModelDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ExternalModelDict":
        return _transform_to_json_value(self)  # type:ignore


class ExternalModelDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    The name of the external model.
    """

    provider: VariableOr[ExternalModelProviderParam]
    """
    The name of the provider for the external model. Currently, the supported providers are 'ai21labs', 'anthropic', 'amazon-bedrock', 'cohere', 'databricks-model-serving', 'google-cloud-vertex-ai', 'openai', 'palm', and 'custom'.
    """

    task: VariableOr[str]
    """
    The task type of the external model.
    """

    ai21labs_config: VariableOrOptional[Ai21LabsConfigParam]
    """
    AI21Labs Config. Only required if the provider is 'ai21labs'.
    """

    amazon_bedrock_config: VariableOrOptional[AmazonBedrockConfigParam]
    """
    Amazon Bedrock Config. Only required if the provider is 'amazon-bedrock'.
    """

    anthropic_config: VariableOrOptional[AnthropicConfigParam]
    """
    Anthropic Config. Only required if the provider is 'anthropic'.
    """

    cohere_config: VariableOrOptional[CohereConfigParam]
    """
    Cohere Config. Only required if the provider is 'cohere'.
    """

    custom_provider_config: VariableOrOptional[CustomProviderConfigParam]
    """
    Custom Provider Config. Only required if the provider is 'custom'.
    """

    databricks_model_serving_config: VariableOrOptional[
        DatabricksModelServingConfigParam
    ]
    """
    Databricks Model Serving Config. Only required if the provider is 'databricks-model-serving'.
    """

    google_cloud_vertex_ai_config: VariableOrOptional[GoogleCloudVertexAiConfigParam]
    """
    Google Cloud Vertex AI Config. Only required if the provider is 'google-cloud-vertex-ai'.
    """

    openai_config: VariableOrOptional[OpenAiConfigParam]
    """
    OpenAI Config. Only required if the provider is 'openai'.
    """

    palm_config: VariableOrOptional[PaLmConfigParam]
    """
    PaLM Config. Only required if the provider is 'palm'.
    """


ExternalModelParam = ExternalModelDict | ExternalModel
