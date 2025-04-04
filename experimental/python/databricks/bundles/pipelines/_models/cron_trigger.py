from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CronTrigger:
    """"""

    quartz_cron_schedule: VariableOrOptional[str] = None

    timezone_id: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "CronTriggerDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CronTriggerDict":
        return _transform_to_json_value(self)  # type:ignore


class CronTriggerDict(TypedDict, total=False):
    """"""

    quartz_cron_schedule: VariableOrOptional[str]

    timezone_id: VariableOrOptional[str]


CronTriggerParam = CronTriggerDict | CronTrigger
