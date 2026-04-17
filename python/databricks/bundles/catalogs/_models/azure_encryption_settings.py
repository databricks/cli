from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AzureEncryptionSettings:
    """
    :meta private: [EXPERIMENTAL]
    """

    azure_tenant_id: VariableOr[str]

    azure_cmk_access_connector_id: VariableOrOptional[str] = None

    azure_cmk_managed_identity_id: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "AzureEncryptionSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AzureEncryptionSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class AzureEncryptionSettingsDict(TypedDict, total=False):
    """"""

    azure_tenant_id: VariableOr[str]

    azure_cmk_access_connector_id: VariableOrOptional[str]

    azure_cmk_managed_identity_id: VariableOrOptional[str]


AzureEncryptionSettingsParam = AzureEncryptionSettingsDict | AzureEncryptionSettings
