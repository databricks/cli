from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GoogleCloudVertexAiConfig:
    """"""

    project_id: VariableOr[str]
    """
    This is the Google Cloud project id that the service account is
    associated with.
    """

    region: VariableOr[str]
    """
    This is the region for the Google Cloud Vertex AI Service. See [supported
    regions] for more details. Some models are only available in specific
    regions.
    
    [supported regions]: https://cloud.google.com/vertex-ai/docs/general/locations
    """

    private_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a private key for the service
    account which has access to the Google Cloud Vertex AI Service. See [Best
    practices for managing service account keys]. If you prefer to paste your
    API key directly, see `private_key_plaintext`. You must provide an API
    key using one of the following fields: `private_key` or
    `private_key_plaintext`
    
    [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
    """

    private_key_plaintext: VariableOrOptional[str] = None
    """
    The private key for the service account which has access to the Google
    Cloud Vertex AI Service provided as a plaintext secret. See [Best
    practices for managing service account keys]. If you prefer to reference
    your key using Databricks Secrets, see `private_key`. You must provide an
    API key using one of the following fields: `private_key` or
    `private_key_plaintext`.
    
    [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
    """

    @classmethod
    def from_dict(cls, value: "GoogleCloudVertexAiConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GoogleCloudVertexAiConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class GoogleCloudVertexAiConfigDict(TypedDict, total=False):
    """"""

    project_id: VariableOr[str]
    """
    This is the Google Cloud project id that the service account is
    associated with.
    """

    region: VariableOr[str]
    """
    This is the region for the Google Cloud Vertex AI Service. See [supported
    regions] for more details. Some models are only available in specific
    regions.
    
    [supported regions]: https://cloud.google.com/vertex-ai/docs/general/locations
    """

    private_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for a private key for the service
    account which has access to the Google Cloud Vertex AI Service. See [Best
    practices for managing service account keys]. If you prefer to paste your
    API key directly, see `private_key_plaintext`. You must provide an API
    key using one of the following fields: `private_key` or
    `private_key_plaintext`
    
    [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
    """

    private_key_plaintext: VariableOrOptional[str]
    """
    The private key for the service account which has access to the Google
    Cloud Vertex AI Service provided as a plaintext secret. See [Best
    practices for managing service account keys]. If you prefer to reference
    your key using Databricks Secrets, see `private_key`. You must provide an
    API key using one of the following fields: `private_key` or
    `private_key_plaintext`.
    
    [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
    """


GoogleCloudVertexAiConfigParam = (
    GoogleCloudVertexAiConfigDict | GoogleCloudVertexAiConfig
)
