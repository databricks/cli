from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList
from databricks.bundles.jobs._models.webhook import Webhook, WebhookParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class WebhookNotifications:
    """"""

    on_duration_warning_threshold_exceeded: VariableOrList[Webhook] = field(
        default_factory=list
    )
    """
    An optional list of system notification IDs to call when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. A maximum of 3 destinations can be specified for the `on_duration_warning_threshold_exceeded` property.
    """

    on_failure: VariableOrList[Webhook] = field(default_factory=list)
    """
    An optional list of system notification IDs to call when the run fails. A maximum of 3 destinations can be specified for the `on_failure` property.
    """

    on_start: VariableOrList[Webhook] = field(default_factory=list)
    """
    An optional list of system notification IDs to call when the run starts. A maximum of 3 destinations can be specified for the `on_start` property.
    """

    on_streaming_backlog_exceeded: VariableOrList[Webhook] = field(default_factory=list)
    """
    An optional list of system notification IDs to call when any streaming backlog thresholds are exceeded for any stream.
    Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
    Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
    A maximum of 3 destinations can be specified for the `on_streaming_backlog_exceeded` property.
    """

    on_success: VariableOrList[Webhook] = field(default_factory=list)
    """
    An optional list of system notification IDs to call when the run completes successfully. A maximum of 3 destinations can be specified for the `on_success` property.
    """

    @classmethod
    def from_dict(cls, value: "WebhookNotificationsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "WebhookNotificationsDict":
        return _transform_to_json_value(self)  # type:ignore


class WebhookNotificationsDict(TypedDict, total=False):
    """"""

    on_duration_warning_threshold_exceeded: VariableOrList[WebhookParam]
    """
    An optional list of system notification IDs to call when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. A maximum of 3 destinations can be specified for the `on_duration_warning_threshold_exceeded` property.
    """

    on_failure: VariableOrList[WebhookParam]
    """
    An optional list of system notification IDs to call when the run fails. A maximum of 3 destinations can be specified for the `on_failure` property.
    """

    on_start: VariableOrList[WebhookParam]
    """
    An optional list of system notification IDs to call when the run starts. A maximum of 3 destinations can be specified for the `on_start` property.
    """

    on_streaming_backlog_exceeded: VariableOrList[WebhookParam]
    """
    An optional list of system notification IDs to call when any streaming backlog thresholds are exceeded for any stream.
    Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
    Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
    A maximum of 3 destinations can be specified for the `on_streaming_backlog_exceeded` property.
    """

    on_success: VariableOrList[WebhookParam]
    """
    An optional list of system notification IDs to call when the run completes successfully. A maximum of 3 destinations can be specified for the `on_success` property.
    """


WebhookNotificationsParam = WebhookNotificationsDict | WebhookNotifications
