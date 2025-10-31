from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.compute_state import (
    ComputeState,
    ComputeStateParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ComputeStatus:
    """"""

    message: VariableOrOptional[str] = None

    state: VariableOrOptional[ComputeState] = None
    """
    State of the app compute.
    """

    @classmethod
    def from_dict(cls, value: "ComputeStatusDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ComputeStatusDict":
        return _transform_to_json_value(self)  # type:ignore


class ComputeStatusDict(TypedDict, total=False):
    """"""

    message: VariableOrOptional[str]

    state: VariableOrOptional[ComputeStateParam]
    """
    State of the app compute.
    """


ComputeStatusParam = ComputeStatusDict | ComputeStatus
