from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.catalogs._models.azure_encryption_settings import (
    AzureEncryptionSettings,
    AzureEncryptionSettingsParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class EncryptionSettings:
    """
    :meta private: [EXPERIMENTAL]

    Encryption Settings are used to carry metadata for securable encryption at rest.
    Currently used for catalogs, we can use the information supplied here to interact with a CMK.
    """

    azure_encryption_settings: VariableOrOptional[AzureEncryptionSettings] = None
    """
    optional Azure settings - only required if an Azure CMK is used.
    """

    azure_key_vault_key_id: VariableOrOptional[str] = None
    """
    the AKV URL in Azure, null otherwise.
    """

    customer_managed_key_id: VariableOrOptional[str] = None
    """
    the CMK uuid in AWS and GCP, null otherwise.
    """

    @classmethod
    def from_dict(cls, value: "EncryptionSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EncryptionSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class EncryptionSettingsDict(TypedDict, total=False):
    """"""

    azure_encryption_settings: VariableOrOptional[AzureEncryptionSettingsParam]
    """
    optional Azure settings - only required if an Azure CMK is used.
    """

    azure_key_vault_key_id: VariableOrOptional[str]
    """
    the AKV URL in Azure, null otherwise.
    """

    customer_managed_key_id: VariableOrOptional[str]
    """
    the CMK uuid in AWS and GCP, null otherwise.
    """


EncryptionSettingsParam = EncryptionSettingsDict | EncryptionSettings
