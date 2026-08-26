from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.jobs._models.continuous_trigger_configuration import (
    ContinuousTriggerConfiguration,
    ContinuousTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.cron_trigger_configuration import (
    CronTriggerConfiguration,
    CronTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.file_arrival_trigger_configuration import (
    FileArrivalTriggerConfiguration,
    FileArrivalTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.model_trigger_configuration import (
    ModelTriggerConfiguration,
    ModelTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam
from databricks.bundles.jobs._models.periodic_trigger_configuration import (
    PeriodicTriggerConfiguration,
    PeriodicTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.sql_condition_configuration import (
    SqlConditionConfiguration,
    SqlConditionConfigurationParam,
)
from databricks.bundles.jobs._models.table_update_trigger_configuration import (
    TableUpdateTriggerConfiguration,
    TableUpdateTriggerConfigurationParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TriggerConfiguration:
    """
    A single trigger attached to a job via `JobSettings.triggers`. Exactly one of the trigger-type fields
    (`periodic`, `schedule`, `continuous`, `file_arrival`, `table_update`, `model`) must be set; mutual exclusivity
    is enforced in the API handler rather than via `oneof` so that codegen, validation, and JSON serialization
    across SDKs and Terraform behave consistently.
    """

    continuous: VariableOrOptional[ContinuousTriggerConfiguration] = None
    """
    [Beta] Continuous trigger configuration.
    """

    file_arrival: VariableOrOptional[FileArrivalTriggerConfiguration] = None
    """
    [Beta] File arrival trigger configuration.
    """

    model: VariableOrOptional[ModelTriggerConfiguration] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Model trigger configuration.
    """

    pause_status: VariableOrOptional[PauseStatus] = None
    """
    [Beta] Whether this trigger is paused. Defaults to UNPAUSED when unset; the server always returns an explicit value on read.
    """

    periodic: VariableOrOptional[PeriodicTriggerConfiguration] = None
    """
    [Beta] Trigger type: exactly one must be set; mutual exclusivity is enforced in the API handler
    Periodic trigger configuration.
    """

    schedule: VariableOrOptional[CronTriggerConfiguration] = None
    """
    [Beta] Cron schedule trigger configuration.
    """

    sql_condition: VariableOrOptional[SqlConditionConfiguration] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional SQL condition that gates whether this trigger fires.
    """

    table_update: VariableOrOptional[TableUpdateTriggerConfiguration] = None
    """
    [Beta] Table update trigger configuration.
    """

    @classmethod
    def from_dict(cls, value: "TriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class TriggerConfigurationDict(TypedDict, total=False):
    """"""

    continuous: VariableOrOptional[ContinuousTriggerConfigurationParam]
    """
    [Beta] Continuous trigger configuration.
    """

    file_arrival: VariableOrOptional[FileArrivalTriggerConfigurationParam]
    """
    [Beta] File arrival trigger configuration.
    """

    model: VariableOrOptional[ModelTriggerConfigurationParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Model trigger configuration.
    """

    pause_status: VariableOrOptional[PauseStatusParam]
    """
    [Beta] Whether this trigger is paused. Defaults to UNPAUSED when unset; the server always returns an explicit value on read.
    """

    periodic: VariableOrOptional[PeriodicTriggerConfigurationParam]
    """
    [Beta] Trigger type: exactly one must be set; mutual exclusivity is enforced in the API handler
    Periodic trigger configuration.
    """

    schedule: VariableOrOptional[CronTriggerConfigurationParam]
    """
    [Beta] Cron schedule trigger configuration.
    """

    sql_condition: VariableOrOptional[SqlConditionConfigurationParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional SQL condition that gates whether this trigger fires.
    """

    table_update: VariableOrOptional[TableUpdateTriggerConfigurationParam]
    """
    [Beta] Table update trigger configuration.
    """


TriggerConfigurationParam = TriggerConfigurationDict | TriggerConfiguration
