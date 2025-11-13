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
class SqlTaskDashboard:
    """"""

    dashboard_id: VariableOr[str]
    """
    The canonical identifier of the SQL dashboard.
    """

    custom_subject: VariableOrOptional[str] = None
    """
    Subject of the email sent to subscribers of this task.
    """

    pause_subscriptions: VariableOrOptional[bool] = None
    """
    If true, the dashboard snapshot is not taken, and emails are not sent to subscribers.
    """

    subscriptions: VariableOrList[SqlTaskSubscription] = field(default_factory=list)
    """
    If specified, dashboard snapshots are sent to subscriptions.
    """

    @classmethod
    def from_dict(cls, value: "SqlTaskDashboardDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SqlTaskDashboardDict":
        return _transform_to_json_value(self)  # type:ignore


class SqlTaskDashboardDict(TypedDict, total=False):
    """"""

    dashboard_id: VariableOr[str]
    """
    The canonical identifier of the SQL dashboard.
    """

    custom_subject: VariableOrOptional[str]
    """
    Subject of the email sent to subscribers of this task.
    """

    pause_subscriptions: VariableOrOptional[bool]
    """
    If true, the dashboard snapshot is not taken, and emails are not sent to subscribers.
    """

    subscriptions: VariableOrList[SqlTaskSubscriptionParam]
    """
    If specified, dashboard snapshots are sent to subscriptions.
    """


SqlTaskDashboardParam = SqlTaskDashboardDict | SqlTaskDashboard
