from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.condition import Condition, ConditionParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class TableUpdateTriggerConfiguration:
    """
    :meta private: [EXPERIMENTAL]
    """

    condition: VariableOrOptional[Condition] = None
    """
    The table(s) condition based on which to trigger a job run.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after the specified amount of time has passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds.
    """

    table_names: VariableOrList[str] = field(default_factory=list)
    """
    A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
    """

    wait_after_last_change_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after no table updates have occurred for the specified time
    and can be used to wait for a series of table updates before triggering a run. The
    minimum allowed value is 60 seconds.
    """

    @classmethod
    def from_dict(cls, value: "TableUpdateTriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TableUpdateTriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class TableUpdateTriggerConfigurationDict(TypedDict, total=False):
    """"""

    condition: VariableOrOptional[ConditionParam]
    """
    The table(s) condition based on which to trigger a job run.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after the specified amount of time has passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds.
    """

    table_names: VariableOrList[str]
    """
    A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
    """

    wait_after_last_change_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after no table updates have occurred for the specified time
    and can be used to wait for a series of table updates before triggering a run. The
    minimum allowed value is 60 seconds.
    """


TableUpdateTriggerConfigurationParam = (
    TableUpdateTriggerConfigurationDict | TableUpdateTriggerConfiguration
)
