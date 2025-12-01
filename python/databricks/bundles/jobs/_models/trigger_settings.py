from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.file_arrival_trigger_configuration import (
    FileArrivalTriggerConfiguration,
    FileArrivalTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam
from databricks.bundles.jobs._models.periodic_trigger_configuration import (
    PeriodicTriggerConfiguration,
    PeriodicTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.table_update_trigger_configuration import (
    TableUpdateTriggerConfiguration,
    TableUpdateTriggerConfigurationParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TriggerSettings:
    """"""

    file_arrival: VariableOrOptional[FileArrivalTriggerConfiguration] = None
    """
    File arrival trigger settings.
    """

    pause_status: VariableOrOptional[PauseStatus] = None
    """
    Whether this trigger is paused or not.
    """

    periodic: VariableOrOptional[PeriodicTriggerConfiguration] = None
    """
    Periodic trigger settings.
    """

    table_update: VariableOrOptional[TableUpdateTriggerConfiguration] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    @classmethod
    def from_dict(cls, value: "TriggerSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TriggerSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class TriggerSettingsDict(TypedDict, total=False):
    """"""

    file_arrival: VariableOrOptional[FileArrivalTriggerConfigurationParam]
    """
    File arrival trigger settings.
    """

    pause_status: VariableOrOptional[PauseStatusParam]
    """
    Whether this trigger is paused or not.
    """

    periodic: VariableOrOptional[PeriodicTriggerConfigurationParam]
    """
    Periodic trigger settings.
    """

    table_update: VariableOrOptional[TableUpdateTriggerConfigurationParam]
    """
    :meta private: [EXPERIMENTAL]
    """


TriggerSettingsParam = TriggerSettingsDict | TriggerSettings
