from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr
from databricks.bundles.jobs._models.periodic_trigger_configuration_time_unit import (
    PeriodicTriggerConfigurationTimeUnit,
    PeriodicTriggerConfigurationTimeUnitParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PeriodicTriggerConfiguration:
    """"""

    interval: VariableOr[int]
    """
    The interval at which the trigger should run.
    """

    unit: VariableOr[PeriodicTriggerConfigurationTimeUnit]
    """
    The unit of time for the interval.
    """

    @classmethod
    def from_dict(cls, value: "PeriodicTriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PeriodicTriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class PeriodicTriggerConfigurationDict(TypedDict, total=False):
    """"""

    interval: VariableOr[int]
    """
    The interval at which the trigger should run.
    """

    unit: VariableOr[PeriodicTriggerConfigurationTimeUnitParam]
    """
    The unit of time for the interval.
    """


PeriodicTriggerConfigurationParam = (
    PeriodicTriggerConfigurationDict | PeriodicTriggerConfiguration
)
