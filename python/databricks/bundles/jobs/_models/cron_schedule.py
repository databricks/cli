from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CronSchedule:
    """"""

    quartz_cron_expression: VariableOr[str]
    """
    A Cron expression using Quartz syntax that describes the schedule for a job. See [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details. This field is required.
    """

    timezone_id: VariableOr[str]
    """
    A Java timezone ID. The schedule for a job is resolved with respect to this timezone. See [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details. This field is required.
    """

    pause_status: VariableOrOptional[PauseStatus] = None
    """
    Indicate whether this schedule is paused or not.
    """

    @classmethod
    def from_dict(cls, value: "CronScheduleDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CronScheduleDict":
        return _transform_to_json_value(self)  # type:ignore


class CronScheduleDict(TypedDict, total=False):
    """"""

    quartz_cron_expression: VariableOr[str]
    """
    A Cron expression using Quartz syntax that describes the schedule for a job. See [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details. This field is required.
    """

    timezone_id: VariableOr[str]
    """
    A Java timezone ID. The schedule for a job is resolved with respect to this timezone. See [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details. This field is required.
    """

    pause_status: VariableOrOptional[PauseStatusParam]
    """
    Indicate whether this schedule is paused or not.
    """


CronScheduleParam = CronScheduleDict | CronSchedule
