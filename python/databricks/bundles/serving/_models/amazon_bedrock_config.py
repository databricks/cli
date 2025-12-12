from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.serving._models.amazon_bedrock_config_bedrock_provider import (
    AmazonBedrockConfigBedrockProvider,
    AmazonBedrockConfigBedrockProviderParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AmazonBedrockConfig:
    """"""

    aws_region: VariableOr[str]
    """
    The AWS region to use. Bedrock has to be enabled there.
    """

    bedrock_provider: VariableOr[AmazonBedrockConfigBedrockProvider]
    """
    The underlying provider in Amazon Bedrock. Supported values (case
    insensitive) include: Anthropic, Cohere, AI21Labs, Amazon.
    """

    aws_access_key_id: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an AWS access key ID with
    permissions to interact with Bedrock services. If you prefer to paste
    your API key directly, see `aws_access_key_id_plaintext`. You must provide an API
    key using one of the following fields: `aws_access_key_id` or
    `aws_access_key_id_plaintext`.
    """

    aws_access_key_id_plaintext: VariableOrOptional[str] = None
    """
    An AWS access key ID with permissions to interact with Bedrock services
    provided as a plaintext string. If you prefer to reference your key using
    Databricks Secrets, see `aws_access_key_id`. You must provide an API key
    using one of the following fields: `aws_access_key_id` or
    `aws_access_key_id_plaintext`.
    """

    aws_secret_access_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an AWS secret access key paired
    with the access key ID, with permissions to interact with Bedrock
    services. If you prefer to paste your API key directly, see
    `aws_secret_access_key_plaintext`. You must provide an API key using one
    of the following fields: `aws_secret_access_key` or
    `aws_secret_access_key_plaintext`.
    """

    aws_secret_access_key_plaintext: VariableOrOptional[str] = None
    """
    An AWS secret access key paired with the access key ID, with permissions
    to interact with Bedrock services provided as a plaintext string. If you
    prefer to reference your key using Databricks Secrets, see
    `aws_secret_access_key`. You must provide an API key using one of the
    following fields: `aws_secret_access_key` or
    `aws_secret_access_key_plaintext`.
    """

    instance_profile_arn: VariableOrOptional[str] = None
    """
    ARN of the instance profile that the external model will use to access AWS resources.
    You must authenticate using an instance profile or access keys.
    If you prefer to authenticate using access keys, see `aws_access_key_id`,
    `aws_access_key_id_plaintext`, `aws_secret_access_key` and `aws_secret_access_key_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "AmazonBedrockConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AmazonBedrockConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class AmazonBedrockConfigDict(TypedDict, total=False):
    """"""

    aws_region: VariableOr[str]
    """
    The AWS region to use. Bedrock has to be enabled there.
    """

    bedrock_provider: VariableOr[AmazonBedrockConfigBedrockProviderParam]
    """
    The underlying provider in Amazon Bedrock. Supported values (case
    insensitive) include: Anthropic, Cohere, AI21Labs, Amazon.
    """

    aws_access_key_id: VariableOrOptional[str]
    """
    The Databricks secret key reference for an AWS access key ID with
    permissions to interact with Bedrock services. If you prefer to paste
    your API key directly, see `aws_access_key_id_plaintext`. You must provide an API
    key using one of the following fields: `aws_access_key_id` or
    `aws_access_key_id_plaintext`.
    """

    aws_access_key_id_plaintext: VariableOrOptional[str]
    """
    An AWS access key ID with permissions to interact with Bedrock services
    provided as a plaintext string. If you prefer to reference your key using
    Databricks Secrets, see `aws_access_key_id`. You must provide an API key
    using one of the following fields: `aws_access_key_id` or
    `aws_access_key_id_plaintext`.
    """

    aws_secret_access_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for an AWS secret access key paired
    with the access key ID, with permissions to interact with Bedrock
    services. If you prefer to paste your API key directly, see
    `aws_secret_access_key_plaintext`. You must provide an API key using one
    of the following fields: `aws_secret_access_key` or
    `aws_secret_access_key_plaintext`.
    """

    aws_secret_access_key_plaintext: VariableOrOptional[str]
    """
    An AWS secret access key paired with the access key ID, with permissions
    to interact with Bedrock services provided as a plaintext string. If you
    prefer to reference your key using Databricks Secrets, see
    `aws_secret_access_key`. You must provide an API key using one of the
    following fields: `aws_secret_access_key` or
    `aws_secret_access_key_plaintext`.
    """

    instance_profile_arn: VariableOrOptional[str]
    """
    ARN of the instance profile that the external model will use to access AWS resources.
    You must authenticate using an instance profile or access keys.
    If you prefer to authenticate using access keys, see `aws_access_key_id`,
    `aws_access_key_id_plaintext`, `aws_secret_access_key` and `aws_secret_access_key_plaintext`.
    """


AmazonBedrockConfigParam = AmazonBedrockConfigDict | AmazonBedrockConfig
