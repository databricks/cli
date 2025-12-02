from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.model_trigger_configuration_condition import (
    ModelTriggerConfigurationCondition,
    ModelTriggerConfigurationConditionParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ModelTriggerConfiguration:
    """
    :meta private: [EXPERIMENTAL]
    """

    condition: VariableOr[ModelTriggerConfigurationCondition]
    """
    The condition based on which to trigger a job run.
    """

    aliases: VariableOrList[str] = field(default_factory=list)
    """
    Aliases of the model versions to monitor. Can only be used in conjunction with condition MODEL_ALIAS_SET.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after the specified amount of time has passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds.
    """

    securable_name: VariableOrOptional[str] = None
    """
    Name of the securable to monitor ("mycatalog.myschema.mymodel" in the case of model-level triggers,
    "mycatalog.myschema" in the case of schema-level triggers) or empty in the case of metastore-level triggers.
    """

    wait_after_last_change_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after no model updates have occurred for the specified time
    and can be used to wait for a series of model updates before triggering a run. The
    minimum allowed value is 60 seconds.
    """

    @classmethod
    def from_dict(cls, value: "ModelTriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ModelTriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class ModelTriggerConfigurationDict(TypedDict, total=False):
    """"""

    condition: VariableOr[ModelTriggerConfigurationConditionParam]
    """
    The condition based on which to trigger a job run.
    """

    aliases: VariableOrList[str]
    """
    Aliases of the model versions to monitor. Can only be used in conjunction with condition MODEL_ALIAS_SET.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after the specified amount of time has passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds.
    """

    securable_name: VariableOrOptional[str]
    """
    Name of the securable to monitor ("mycatalog.myschema.mymodel" in the case of model-level triggers,
    "mycatalog.myschema" in the case of schema-level triggers) or empty in the case of metastore-level triggers.
    """

    wait_after_last_change_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after no model updates have occurred for the specified time
    and can be used to wait for a series of model updates before triggering a run. The
    minimum allowed value is 60 seconds.
    """


ModelTriggerConfigurationParam = (
    ModelTriggerConfigurationDict | ModelTriggerConfiguration
)
