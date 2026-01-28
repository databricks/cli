from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.model_serving_endpoints._models.api_key_auth import (
    ApiKeyAuth,
    ApiKeyAuthParam,
)
from databricks.bundles.model_serving_endpoints._models.bearer_token_auth import (
    BearerTokenAuth,
    BearerTokenAuthParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CustomProviderConfig:
    """
    Configs needed to create a custom provider model route.
    """

    custom_provider_url: VariableOr[str]
    """
    This is a field to provide the URL of the custom provider API.
    """

    api_key_auth: VariableOrOptional[ApiKeyAuth] = None
    """
    This is a field to provide API key authentication for the custom provider API.
    You can only specify one authentication method.
    """

    bearer_token_auth: VariableOrOptional[BearerTokenAuth] = None
    """
    This is a field to provide bearer token authentication for the custom provider API.
    You can only specify one authentication method.
    """

    @classmethod
    def from_dict(cls, value: "CustomProviderConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CustomProviderConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class CustomProviderConfigDict(TypedDict, total=False):
    """"""

    custom_provider_url: VariableOr[str]
    """
    This is a field to provide the URL of the custom provider API.
    """

    api_key_auth: VariableOrOptional[ApiKeyAuthParam]
    """
    This is a field to provide API key authentication for the custom provider API.
    You can only specify one authentication method.
    """

    bearer_token_auth: VariableOrOptional[BearerTokenAuthParam]
    """
    This is a field to provide bearer token authentication for the custom provider API.
    You can only specify one authentication method.
    """


CustomProviderConfigParam = CustomProviderConfigDict | CustomProviderConfig
