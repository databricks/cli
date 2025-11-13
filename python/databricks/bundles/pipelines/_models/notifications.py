from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Notifications:
    """"""

    alerts: VariableOrList[str] = field(default_factory=list)
    """
    A list of alerts that trigger the sending of notifications to the configured
    destinations. The supported alerts are:
    
    * `on-update-success`: A pipeline update completes successfully.
    * `on-update-failure`: Each time a pipeline update fails.
    * `on-update-fatal-failure`: A pipeline update fails with a non-retryable (fatal) error.
    * `on-flow-failure`: A single data flow fails.
    """

    email_recipients: VariableOrList[str] = field(default_factory=list)
    """
    A list of email addresses notified when a configured alert is triggered.
    """

    @classmethod
    def from_dict(cls, value: "NotificationsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "NotificationsDict":
        return _transform_to_json_value(self)  # type:ignore


class NotificationsDict(TypedDict, total=False):
    """"""

    alerts: VariableOrList[str]
    """
    A list of alerts that trigger the sending of notifications to the configured
    destinations. The supported alerts are:
    
    * `on-update-success`: A pipeline update completes successfully.
    * `on-update-failure`: Each time a pipeline update fails.
    * `on-update-fatal-failure`: A pipeline update fails with a non-retryable (fatal) error.
    * `on-flow-failure`: A single data flow fails.
    """

    email_recipients: VariableOrList[str]
    """
    A list of email addresses notified when a configured alert is triggered.
    """


NotificationsParam = NotificationsDict | Notifications
