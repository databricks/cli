from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrDict, VariableOrOptional
from databricks.bundles.jobs._models.subscription import Subscription, SubscriptionParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DashboardTask:
    """
    Configures the Lakeview Dashboard job task type.
    """

    dashboard_id: VariableOrOptional[str] = None

    filters: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    Dashboard task parameters. Used to apply dashboard filter values during dashboard task execution. Parameter values get applied to any dashboard filters that have a matching URL identifier as the parameter key.
    The parameter value format is dependent on the filter type:
    - For text and single-select filters, provide a single value (e.g. `"value"`)
    - For date and datetime filters, provide the value in ISO 8601 format (e.g. `"2000-01-01T00:00:00"`)
    - For multi-select filters, provide a JSON array of values (e.g. `"[\"value1\",\"value2\"]"`)
    - For range and date range filters, provide a JSON object with `start` and `end` (e.g. `"{\"start\":\"1\",\"end\":\"10\"}"`)
    """

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

    filters: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Dashboard task parameters. Used to apply dashboard filter values during dashboard task execution. Parameter values get applied to any dashboard filters that have a matching URL identifier as the parameter key.
    The parameter value format is dependent on the filter type:
    - For text and single-select filters, provide a single value (e.g. `"value"`)
    - For date and datetime filters, provide the value in ISO 8601 format (e.g. `"2000-01-01T00:00:00"`)
    - For multi-select filters, provide a JSON array of values (e.g. `"[\"value1\",\"value2\"]"`)
    - For range and date range filters, provide a JSON object with `start` and `end` (e.g. `"{\"start\":\"1\",\"end\":\"10\"}"`)
    """

    subscription: VariableOrOptional[SubscriptionParam]

    warehouse_id: VariableOrOptional[str]
    """
    Optional: The warehouse id to execute the dashboard with for the schedule.
    If not specified, the default warehouse of the dashboard will be used.
    """


DashboardTaskParam = DashboardTaskDict | DashboardTask
