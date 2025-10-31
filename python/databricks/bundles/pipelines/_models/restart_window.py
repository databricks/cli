from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.day_of_week import DayOfWeek, DayOfWeekParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class RestartWindow:
    """
    :meta private: [EXPERIMENTAL]
    """

    start_hour: VariableOr[int]
    """
    An integer between 0 and 23 denoting the start hour for the restart window in the 24-hour day.
    Continuous pipeline restart is triggered only within a five-hour window starting at this hour.
    """

    days_of_week: VariableOrList[DayOfWeek] = field(default_factory=list)
    """
    Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour).
    If not specified all days of the week will be used.
    """

    time_zone_id: VariableOrOptional[str] = None
    """
    Time zone id of restart window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details.
    If not specified, UTC will be used.
    """

    @classmethod
    def from_dict(cls, value: "RestartWindowDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RestartWindowDict":
        return _transform_to_json_value(self)  # type:ignore


class RestartWindowDict(TypedDict, total=False):
    """"""

    start_hour: VariableOr[int]
    """
    An integer between 0 and 23 denoting the start hour for the restart window in the 24-hour day.
    Continuous pipeline restart is triggered only within a five-hour window starting at this hour.
    """

    days_of_week: VariableOrList[DayOfWeekParam]
    """
    Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour).
    If not specified all days of the week will be used.
    """

    time_zone_id: VariableOrOptional[str]
    """
    Time zone id of restart window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details.
    If not specified, UTC will be used.
    """


RestartWindowParam = RestartWindowDict | RestartWindow
