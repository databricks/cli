from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Continuous:
    """"""

    pause_status: VariableOrOptional[PauseStatus] = None
    """
    Indicate whether the continuous execution of the job is paused or not. Defaults to UNPAUSED.
    """

    @classmethod
    def from_dict(cls, value: "ContinuousDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ContinuousDict":
        return _transform_to_json_value(self)  # type:ignore


class ContinuousDict(TypedDict, total=False):
    """"""

    pause_status: VariableOrOptional[PauseStatusParam]
    """
    Indicate whether the continuous execution of the job is paused or not. Defaults to UNPAUSED.
    """


ContinuousParam = ContinuousDict | Continuous
