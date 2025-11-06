from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TaskNotificationSettings:
    """"""

    alert_on_last_attempt: VariableOrOptional[bool] = None
    """
    If true, do not send notifications to recipients specified in `on_start` for the retried runs and do not send notifications to recipients specified in `on_failure` until the last retry of the run.
    """

    no_alert_for_canceled_runs: VariableOrOptional[bool] = None
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
    """

    no_alert_for_skipped_runs: VariableOrOptional[bool] = None
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
    """

    @classmethod
    def from_dict(cls, value: "TaskNotificationSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TaskNotificationSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class TaskNotificationSettingsDict(TypedDict, total=False):
    """"""

    alert_on_last_attempt: VariableOrOptional[bool]
    """
    If true, do not send notifications to recipients specified in `on_start` for the retried runs and do not send notifications to recipients specified in `on_failure` until the last retry of the run.
    """

    no_alert_for_canceled_runs: VariableOrOptional[bool]
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
    """

    no_alert_for_skipped_runs: VariableOrOptional[bool]
    """
    If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
    """


TaskNotificationSettingsParam = TaskNotificationSettingsDict | TaskNotificationSettings
