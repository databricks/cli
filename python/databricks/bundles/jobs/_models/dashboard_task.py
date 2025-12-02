from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.subscription import (
    Subscription,
    SubscriptionParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DashboardTask:
    """
    Configures the Lakeview Dashboard job task type.
    """

    dashboard_id: VariableOrOptional[str] = None

    subscription: VariableOrOptional[Subscription] = None

    warehouse_id: VariableOrOptional[str] = None
    """
    Optional: The warehouse id to execute the dashboard with for the schedule.
    If not specified, the default warehouse of the dashboard will be used.
    """

    @classmethod
    def from_dict(cls, value: "DashboardTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DashboardTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class DashboardTaskDict(TypedDict, total=False):
    """"""

    dashboard_id: VariableOrOptional[str]

    subscription: VariableOrOptional[SubscriptionParam]

    warehouse_id: VariableOrOptional[str]
    """
    Optional: The warehouse id to execute the dashboard with for the schedule.
    If not specified, the default warehouse of the dashboard will be used.
    """


DashboardTaskParam = DashboardTaskDict | DashboardTask
