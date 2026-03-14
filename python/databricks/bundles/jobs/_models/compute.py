from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.hardware_accelerator_type import (
    HardwareAcceleratorType,
    HardwareAcceleratorTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Compute:
    """"""

    hardware_accelerator: VariableOrOptional[HardwareAcceleratorType] = None
    """
    Hardware accelerator configuration for Serverless GPU workloads.
    """

    @classmethod
    def from_dict(cls, value: "ComputeDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ComputeDict":
        return _transform_to_json_value(self)  # type:ignore


class ComputeDict(TypedDict, total=False):
    """"""

    hardware_accelerator: VariableOrOptional[HardwareAcceleratorTypeParam]
    """
    Hardware accelerator configuration for Serverless GPU workloads.
    """


ComputeParam = ComputeDict | Compute
