from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GoogleAdsOptions:
    """
    :meta private: [EXPERIMENTAL]

    Google Ads specific options for ingestion (object-level).
    When set, these values override the corresponding fields in GoogleAdsConfig
    (source_configurations).
    """

    manager_account_id: VariableOr[str]
    """
    (Optional at this level) Manager Account ID (also called MCC Account ID) used to list
    and access customer accounts under this manager account.
    Overrides GoogleAdsConfig.manager_account_id from source_configurations when set.
    """

    lookback_window_days: VariableOrOptional[int] = None
    """
    (Optional) Number of days to look back for report tables to capture late-arriving data.
    If not specified, defaults to 30 days.
    """

    sync_start_date: VariableOrOptional[str] = None
    """
    (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, defaults to 2 years of historical data.
    """

    @classmethod
    def from_dict(cls, value: "GoogleAdsOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GoogleAdsOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class GoogleAdsOptionsDict(TypedDict, total=False):
    """"""

    manager_account_id: VariableOr[str]
    """
    (Optional at this level) Manager Account ID (also called MCC Account ID) used to list
    and access customer accounts under this manager account.
    Overrides GoogleAdsConfig.manager_account_id from source_configurations when set.
    """

    lookback_window_days: VariableOrOptional[int]
    """
    (Optional) Number of days to look back for report tables to capture late-arriving data.
    If not specified, defaults to 30 days.
    """

    sync_start_date: VariableOrOptional[str]
    """
    (Optional) Start date for the initial sync of report tables in YYYY-MM-DD format.
    This determines the earliest date from which to sync historical data.
    If not specified, defaults to 2 years of historical data.
    """


GoogleAdsOptionsParam = GoogleAdsOptionsDict | GoogleAdsOptions
