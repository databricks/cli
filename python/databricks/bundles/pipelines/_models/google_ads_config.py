from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GoogleAdsConfig:
    """
    :meta private: [EXPERIMENTAL]
    """

    manager_account_id: VariableOrOptional[str] = None
    """
    (Required) Manager Account ID (also called MCC Account ID) used to list and access
    customer accounts under this manager account. This is required for fetching the list
    of customer accounts during source selection.
    If the same field is also set in the object-level GoogleAdsOptions (connector_options),
    the object-level value takes precedence over this top-level config.
    """

    @classmethod
    def from_dict(cls, value: "GoogleAdsConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GoogleAdsConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class GoogleAdsConfigDict(TypedDict, total=False):
    """"""

    manager_account_id: VariableOrOptional[str]
    """
    (Required) Manager Account ID (also called MCC Account ID) used to list and access
    customer accounts under this manager account. This is required for fetching the list
    of customer accounts during source selection.
    If the same field is also set in the object-level GoogleAdsOptions (connector_options),
    the object-level value takes precedence over this top-level config.
    """


GoogleAdsConfigParam = GoogleAdsConfigDict | GoogleAdsConfig
