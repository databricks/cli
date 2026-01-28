from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DatabricksModelServingConfig:
    """"""

    databricks_workspace_url: VariableOr[str]
    """
    The URL of the Databricks workspace containing the model serving endpoint
    pointed to by this external model.
    """

    databricks_api_token: VariableOrOptional[str] = None
    """
    The Databricks secret key reference for a Databricks API token that
    corresponds to a user or service principal with Can Query access to the
    model serving endpoint pointed to by this external model. If you prefer
    to paste your API key directly, see `databricks_api_token_plaintext`. You
    must provide an API key using one of the following fields:
    `databricks_api_token` or `databricks_api_token_plaintext`.
    """

    databricks_api_token_plaintext: VariableOrOptional[str] = None
    """
    The Databricks API token that corresponds to a user or service principal
    with Can Query access to the model serving endpoint pointed to by this
    external model provided as a plaintext string. If you prefer to reference
    your key using Databricks Secrets, see `databricks_api_token`. You must
    provide an API key using one of the following fields:
    `databricks_api_token` or `databricks_api_token_plaintext`.
    """

    @classmethod
    def from_dict(cls, value: "DatabricksModelServingConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DatabricksModelServingConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class DatabricksModelServingConfigDict(TypedDict, total=False):
    """"""

    databricks_workspace_url: VariableOr[str]
    """
    The URL of the Databricks workspace containing the model serving endpoint
    pointed to by this external model.
    """

    databricks_api_token: VariableOrOptional[str]
    """
    The Databricks secret key reference for a Databricks API token that
    corresponds to a user or service principal with Can Query access to the
    model serving endpoint pointed to by this external model. If you prefer
    to paste your API key directly, see `databricks_api_token_plaintext`. You
    must provide an API key using one of the following fields:
    `databricks_api_token` or `databricks_api_token_plaintext`.
    """

    databricks_api_token_plaintext: VariableOrOptional[str]
    """
    The Databricks API token that corresponds to a user or service principal
    with Can Query access to the model serving endpoint pointed to by this
    external model provided as a plaintext string. If you prefer to reference
    your key using Databricks Secrets, see `databricks_api_token`. You must
    provide an API key using one of the following fields:
    `databricks_api_token` or `databricks_api_token_plaintext`.
    """


DatabricksModelServingConfigParam = (
    DatabricksModelServingConfigDict | DatabricksModelServingConfig
)
