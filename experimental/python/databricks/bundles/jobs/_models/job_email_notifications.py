from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobEmailNotifications:
    """"""

    on_duration_warning_threshold_exceeded: VariableOrList[str] = field(
        default_factory=list
    )
    """
    A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
    """

    on_failure: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
    """

    on_start: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
    """

    on_streaming_backlog_exceeded: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream.
    Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
    Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
    """

    on_success: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
    """

    @classmethod
    def from_dict(cls, value: "JobEmailNotificationsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobEmailNotificationsDict":
        return _transform_to_json_value(self)  # type:ignore


class JobEmailNotificationsDict(TypedDict, total=False):
    """"""

    on_duration_warning_threshold_exceeded: VariableOrList[str]
    """
    A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
    """

    on_failure: VariableOrList[str]
    """
    A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
    """

    on_start: VariableOrList[str]
    """
    A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
    """

    on_streaming_backlog_exceeded: VariableOrList[str]
    """
    A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream.
    Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
    Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
    """

    on_success: VariableOrList[str]
    """
    A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
    """


JobEmailNotificationsParam = JobEmailNotificationsDict | JobEmailNotifications
