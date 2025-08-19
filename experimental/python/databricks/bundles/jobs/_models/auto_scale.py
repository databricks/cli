from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AutoScale:
    """"""

    max_workers: VariableOrOptional[int] = None
    """
    The maximum number of workers to which the cluster can scale up when overloaded.
    Note that `max_workers` must be strictly greater than `min_workers`.
    """

    min_workers: VariableOrOptional[int] = None
    """
    The minimum number of workers to which the cluster can scale down when underutilized.
    It is also the initial number of workers the cluster will have after creation.
    """

    @classmethod
    def from_dict(cls, value: "AutoScaleDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AutoScaleDict":
        return _transform_to_json_value(self)  # type:ignore


class AutoScaleDict(TypedDict, total=False):
    """"""

    max_workers: VariableOrOptional[int]
    """
    The maximum number of workers to which the cluster can scale up when overloaded.
    Note that `max_workers` must be strictly greater than `min_workers`.
    """

    min_workers: VariableOrOptional[int]
    """
    The minimum number of workers to which the cluster can scale down when underutilized.
    It is also the initial number of workers the cluster will have after creation.
    """


AutoScaleParam = AutoScaleDict | AutoScale
