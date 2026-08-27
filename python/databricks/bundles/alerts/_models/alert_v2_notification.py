from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.alerts._models.alert_v2_subscription import (
    AlertV2Subscription,
    AlertV2SubscriptionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertV2Notification:
    """"""

    notify_on_ok: VariableOrOptional[bool] = None
    """
    Whether to notify alert subscribers when alert returns back to normal.
    """

    retrigger_seconds: VariableOrOptional[int] = None
    """
    Number of seconds an alert waits after being triggered before it is allowed to send another notification.
    If set to 0 or omitted, the alert will not send any further notifications after the first trigger
    Setting this value to 1 allows the alert to send a notification on every evaluation where the condition is met, effectively making it always retrigger for notification purposes.
    """

    subscriptions: VariableOrList[AlertV2Subscription] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "AlertV2NotificationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertV2NotificationDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertV2NotificationDict(TypedDict, total=False):
    """"""

    notify_on_ok: VariableOrOptional[bool]
    """
    Whether to notify alert subscribers when alert returns back to normal.
    """

    retrigger_seconds: VariableOrOptional[int]
    """
    Number of seconds an alert waits after being triggered before it is allowed to send another notification.
    If set to 0 or omitted, the alert will not send any further notifications after the first trigger
    Setting this value to 1 allows the alert to send a notification on every evaluation where the condition is met, effectively making it always retrigger for notification purposes.
    """

    subscriptions: VariableOrList[AlertV2SubscriptionParam]


AlertV2NotificationParam = AlertV2NotificationDict | AlertV2Notification
