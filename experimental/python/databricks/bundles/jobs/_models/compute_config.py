from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ComputeConfig:
    """
    :meta private: [EXPERIMENTAL]
    """

    num_gpus: VariableOr[int]
    """
    Number of GPUs.
    """

    gpu_node_pool_id: VariableOrOptional[str] = None
    """
    IDof the GPU pool to use.
    """

    gpu_type: VariableOrOptional[str] = None
    """
    GPU type.
    """

    @classmethod
    def from_dict(cls, value: "ComputeConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ComputeConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class ComputeConfigDict(TypedDict, total=False):
    """"""

    num_gpus: VariableOr[int]
    """
    Number of GPUs.
    """

    gpu_node_pool_id: VariableOrOptional[str]
    """
    IDof the GPU pool to use.
    """

    gpu_type: VariableOrOptional[str]
    """
    GPU type.
    """


ComputeConfigParam = ComputeConfigDict | ComputeConfig
