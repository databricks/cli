from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CronTriggerConfiguration:
    """
    Cron schedule trigger. Stripped-down counterpart to `CronSchedule`: `pause_status` and `sql_condition` are owned
    by the enclosing `TriggerConfiguration` and intentionally omitted here.
    """

    quartz_cron_expression: VariableOr[str]
    """
    [Beta] A Cron expression using Quartz syntax that describes the schedule for this trigger. See
    [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details.
    """

    timezone_id: VariableOr[str]
    """
    [Beta] A Java timezone ID. The schedule is resolved with respect to this timezone. See
    [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details.
    """

    @classmethod
    def from_dict(cls, value: "CronTriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CronTriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class CronTriggerConfigurationDict(TypedDict, total=False):
    """"""

    quartz_cron_expression: VariableOr[str]
    """
    [Beta] A Cron expression using Quartz syntax that describes the schedule for this trigger. See
    [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details.
    """

    timezone_id: VariableOr[str]
    """
    [Beta] A Java timezone ID. The schedule is resolved with respect to this timezone. See
    [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details.
    """


CronTriggerConfigurationParam = CronTriggerConfigurationDict | CronTriggerConfiguration
