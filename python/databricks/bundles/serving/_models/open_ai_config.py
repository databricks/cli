from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class OpenAiConfig:
    """
    Configs needed to create an OpenAI model route.
    """

    microsoft_entra_client_id: VariableOrOptional[str] = None
    """
    This field is only required for Azure AD OpenAI and is the Microsoft
    Entra Client ID.
    """

    microsoft_entra_client_secret: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a client secret used for
    Microsoft Entra ID authentication. If you prefer to paste your client
    secret directly, see `microsoft_entra_client_secret_plaintext`. You must
    provide an API key using one of the following fields:
    `microsoft_entra_client_secret` or
    `microsoft_entra_client_secret_plaintext`.
    """

    microsoft_entra_client_secret_plaintext: VariableOrOptional[str] = None
    """
    The client secret used for Microsoft Entra ID authentication provided as
    a plaintext string. If you prefer to reference your key using Databricks
    Secrets, see `microsoft_entra_client_secret`. You must provide an API key
    using one of the following fields: `microsoft_entra_client_secret` or
    `microsoft_entra_client_secret_plaintext`.
    """

    microsoft_entra_tenant_id: VariableOrOptional[str] = None
    """
    This field is only required for Azure AD OpenAI and is the Microsoft
    Entra Tenant ID.
    """

    openai_api_base: VariableOrOptional[str] = None
    """
    This is a field to provide a customized base URl for the OpenAI API. For
    Azure OpenAI, this field is required, and is the base URL for the Azure
    OpenAI API service provided by Azure. For other OpenAI API types, this
    field is optional, and if left unspecified, the standard OpenAI base URL
    is used.
    """

    openai_api_key: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for an OpenAI API key using the
    OpenAI or Azure service. If you prefer to paste your API key directly,
    see `openai_api_key_plaintext`. You must provide an API key using one of
    the following fields: `openai_api_key` or `openai_api_key_plaintext`.
    """

    openai_api_key_plaintext: VariableOrOptional[str] = None
    """
    The OpenAI API key using the OpenAI or Azure service provided as a
    plaintext string. If you prefer to reference your key using Databricks
    Secrets, see `openai_api_key`. You must provide an API key using one of
    the following fields: `openai_api_key` or `openai_api_key_plaintext`.
    """

    openai_api_type: VariableOrOptional[str] = None
    """
    This is an optional field to specify the type of OpenAI API to use. For
    Azure OpenAI, this field is required, and adjust this parameter to
    represent the preferred security access validation protocol. For access
    token validation, use azure. For authentication using Azure Active
    Directory (Azure AD) use, azuread.
    """

    openai_api_version: VariableOrOptional[str] = None
    """
    This is an optional field to specify the OpenAI API version. For Azure
    OpenAI, this field is required, and is the version of the Azure OpenAI
    service to utilize, specified by a date.
    """

    openai_deployment_name: VariableOrOptional[str] = None
    """
    This field is only required for Azure OpenAI and is the name of the
    deployment resource for the Azure OpenAI service.
    """

    openai_organization: VariableOrOptional[str] = None
    """
    This is an optional field to specify the organization in OpenAI or Azure
    OpenAI.
    """

    @classmethod
    def from_dict(cls, value: "OpenAiConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "OpenAiConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class OpenAiConfigDict(TypedDict, total=False):
    """"""

    microsoft_entra_client_id: VariableOrOptional[str]
    """
    This field is only required for Azure AD OpenAI and is the Microsoft
    Entra Client ID.
    """

    microsoft_entra_client_secret: VariableOrOptional[str]
    """
    The Databricks secret key reference for a client secret used for
    Microsoft Entra ID authentication. If you prefer to paste your client
    secret directly, see `microsoft_entra_client_secret_plaintext`. You must
    provide an API key using one of the following fields:
    `microsoft_entra_client_secret` or
    `microsoft_entra_client_secret_plaintext`.
    """

    microsoft_entra_client_secret_plaintext: VariableOrOptional[str]
    """
    The client secret used for Microsoft Entra ID authentication provided as
    a plaintext string. If you prefer to reference your key using Databricks
    Secrets, see `microsoft_entra_client_secret`. You must provide an API key
    using one of the following fields: `microsoft_entra_client_secret` or
    `microsoft_entra_client_secret_plaintext`.
    """

    microsoft_entra_tenant_id: VariableOrOptional[str]
    """
    This field is only required for Azure AD OpenAI and is the Microsoft
    Entra Tenant ID.
    """

    openai_api_base: VariableOrOptional[str]
    """
    This is a field to provide a customized base URl for the OpenAI API. For
    Azure OpenAI, this field is required, and is the base URL for the Azure
    OpenAI API service provided by Azure. For other OpenAI API types, this
    field is optional, and if left unspecified, the standard OpenAI base URL
    is used.
    """

    openai_api_key: VariableOrOptional[str]
    """
    The Databricks secret key reference for an OpenAI API key using the
    OpenAI or Azure service. If you prefer to paste your API key directly,
    see `openai_api_key_plaintext`. You must provide an API key using one of
    the following fields: `openai_api_key` or `openai_api_key_plaintext`.
    """

    openai_api_key_plaintext: VariableOrOptional[str]
    """
    The OpenAI API key using the OpenAI or Azure service provided as a
    plaintext string. If you prefer to reference your key using Databricks
    Secrets, see `openai_api_key`. You must provide an API key using one of
    the following fields: `openai_api_key` or `openai_api_key_plaintext`.
    """

    openai_api_type: VariableOrOptional[str]
    """
    This is an optional field to specify the type of OpenAI API to use. For
    Azure OpenAI, this field is required, and adjust this parameter to
    represent the preferred security access validation protocol. For access
    token validation, use azure. For authentication using Azure Active
    Directory (Azure AD) use, azuread.
    """

    openai_api_version: VariableOrOptional[str]
    """
    This is an optional field to specify the OpenAI API version. For Azure
    OpenAI, this field is required, and is the version of the Azure OpenAI
    service to utilize, specified by a date.
    """

    openai_deployment_name: VariableOrOptional[str]
    """
    This field is only required for Azure OpenAI and is the name of the
    deployment resource for the Azure OpenAI service.
    """

    openai_organization: VariableOrOptional[str]
    """
    This is an optional field to specify the organization in OpenAI or Azure
    OpenAI.
    """


OpenAiConfigParam = OpenAiConfigDict | OpenAiConfig
