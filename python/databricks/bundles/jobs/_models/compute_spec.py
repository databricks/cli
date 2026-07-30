from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr
from databricks.bundles.jobs._models.compute_spec_accelerator_type import (
    ComputeSpecAcceleratorType,
    ComputeSpecAcceleratorTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ComputeSpec:
    """
    :meta private: [EXPERIMENTAL]

    ComputeSpec: compute configuration — accelerator type and total
    accelerator count across all nodes.
    """

    accelerator_count: VariableOr[int]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Total number of accelerators across all nodes. Must be a positive
    multiple of the per-node accelerator count encoded in `accelerator_type`.
    For example, `GPU_8xH100` with `accelerator_count: 16` allocates 2 nodes
    (8 GPUs per node).
    """

    accelerator_type: VariableOr[ComputeSpecAcceleratorType]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Hardware accelerator type (for example, `GPU_1xA10` or `GPU_8xH100`).
    The number of accelerators per node is encoded in the enum value —
    `GPU_8xH100` means 8 H100 GPUs per node.
    """

    @classmethod
    def from_dict(cls, value: "ComputeSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ComputeSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class ComputeSpecDict(TypedDict, total=False):
    """"""

    accelerator_count: VariableOr[int]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Total number of accelerators across all nodes. Must be a positive
    multiple of the per-node accelerator count encoded in `accelerator_type`.
    For example, `GPU_8xH100` with `accelerator_count: 16` allocates 2 nodes
    (8 GPUs per node).
    """

    accelerator_type: VariableOr[ComputeSpecAcceleratorTypeParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Hardware accelerator type (for example, `GPU_1xA10` or `GPU_8xH100`).
    The number of accelerators per node is encoded in the enum value —
    `GPU_8xH100` means 8 H100 GPUs per node.
    """


ComputeSpecParam = ComputeSpecDict | ComputeSpec
