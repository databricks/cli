from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.cron_trigger import (
    CronTrigger,
    CronTriggerParam,
)
from databricks.bundles.pipelines._models.manual_trigger import (
    ManualTrigger,
    ManualTriggerParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineTrigger:
    """"""

    cron: VariableOrOptional[CronTrigger] = None

    manual: VariableOrOptional[ManualTrigger] = None

    @classmethod
    def from_dict(cls, value: "PipelineTriggerDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineTriggerDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineTriggerDict(TypedDict, total=False):
    """"""

    cron: VariableOrOptional[CronTriggerParam]

    manual: VariableOrOptional[ManualTriggerParam]


PipelineTriggerParam = PipelineTriggerDict | PipelineTrigger
