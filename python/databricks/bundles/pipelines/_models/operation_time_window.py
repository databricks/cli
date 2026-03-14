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
class OperationTimeWindow:
    """
    Proto representing a window
    """

    start_hour: VariableOr[int]
    """
    An integer between 0 and 23 denoting the start hour for the window in the 24-hour day.
    """

    days_of_week: VariableOrList[DayOfWeek] = field(default_factory=list)
    """
    Days of week in which the window is allowed to happen
    If not specified all days of the week will be used.
    """

    time_zone_id: VariableOrOptional[str] = None
    """
    Time zone id of window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details.
    If not specified, UTC will be used.
    """

    @classmethod
    def from_dict(cls, value: "OperationTimeWindowDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "OperationTimeWindowDict":
        return _transform_to_json_value(self)  # type:ignore


class OperationTimeWindowDict(TypedDict, total=False):
    """"""

    start_hour: VariableOr[int]
    """
    An integer between 0 and 23 denoting the start hour for the window in the 24-hour day.
    """

    days_of_week: VariableOrList[DayOfWeekParam]
    """
    Days of week in which the window is allowed to happen
    If not specified all days of the week will be used.
    """

    time_zone_id: VariableOrOptional[str]
    """
    Time zone id of window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details.
    If not specified, UTC will be used.
    """


OperationTimeWindowParam = OperationTimeWindowDict | OperationTimeWindow
