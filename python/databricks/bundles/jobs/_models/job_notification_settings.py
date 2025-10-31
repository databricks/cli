from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobNotificationSettings:
    """"""

    no_alert_for_canceled_runs: VariableOrOptional[bool] = None
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
    """

    no_alert_for_skipped_runs: VariableOrOptional[bool] = None
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
    """

    @classmethod
    def from_dict(cls, value: "JobNotificationSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobNotificationSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class JobNotificationSettingsDict(TypedDict, total=False):
    """"""

    no_alert_for_canceled_runs: VariableOrOptional[bool]
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
    """

    no_alert_for_skipped_runs: VariableOrOptional[bool]
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
    """


JobNotificationSettingsParam = JobNotificationSettingsDict | JobNotificationSettings
