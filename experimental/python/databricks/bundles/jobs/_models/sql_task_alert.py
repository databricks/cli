from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.sql_task_subscription import (
    SqlTaskSubscription,
    SqlTaskSubscriptionParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SqlTaskAlert:
    """"""

    alert_id: VariableOr[str]
    """
    The canonical identifier of the SQL alert.
    """

    pause_subscriptions: VariableOrOptional[bool] = None
    """
    If true, the alert notifications are not sent to subscribers.
    """

    subscriptions: VariableOrList[SqlTaskSubscription] = field(default_factory=list)
    """
    If specified, alert notifications are sent to subscribers.
    """

    @classmethod
    def from_dict(cls, value: "SqlTaskAlertDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SqlTaskAlertDict":
        return _transform_to_json_value(self)  # type:ignore


class SqlTaskAlertDict(TypedDict, total=False):
    """"""

    alert_id: VariableOr[str]
    """
    The canonical identifier of the SQL alert.
    """

    pause_subscriptions: VariableOrOptional[bool]
    """
    If true, the alert notifications are not sent to subscribers.
    """

    subscriptions: VariableOrList[SqlTaskSubscriptionParam]
    """
    If specified, alert notifications are sent to subscribers.
    """


SqlTaskAlertParam = SqlTaskAlertDict | SqlTaskAlert
