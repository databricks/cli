from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ManualTrigger:
    """"""

    @classmethod
    def from_dict(cls, value: "ManualTriggerDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ManualTriggerDict":
        return _transform_to_json_value(self)  # type:ignore


class ManualTriggerDict(TypedDict, total=False):
    """"""


ManualTriggerParam = ManualTriggerDict | ManualTrigger
