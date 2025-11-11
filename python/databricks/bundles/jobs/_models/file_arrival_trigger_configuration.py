from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class FileArrivalTriggerConfiguration:
    """"""

    url: VariableOr[str]
    """
    URL to be monitored for file arrivals. The path must point to the root or a subpath of the external location.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after the specified amount of time passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds
    """

    wait_after_last_change_seconds: VariableOrOptional[int] = None
    """
    If set, the trigger starts a run only after no file activity has occurred for the specified amount of time.
    This makes it possible to wait for a batch of incoming files to arrive before triggering a run. The
    minimum allowed value is 60 seconds.
    """

    @classmethod
    def from_dict(cls, value: "FileArrivalTriggerConfigurationDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FileArrivalTriggerConfigurationDict":
        return _transform_to_json_value(self)  # type:ignore


class FileArrivalTriggerConfigurationDict(TypedDict, total=False):
    """"""

    url: VariableOr[str]
    """
    URL to be monitored for file arrivals. The path must point to the root or a subpath of the external location.
    """

    min_time_between_triggers_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after the specified amount of time passed since
    the last time the trigger fired. The minimum allowed value is 60 seconds
    """

    wait_after_last_change_seconds: VariableOrOptional[int]
    """
    If set, the trigger starts a run only after no file activity has occurred for the specified amount of time.
    This makes it possible to wait for a batch of incoming files to arrive before triggering a run. The
    minimum allowed value is 60 seconds.
    """


FileArrivalTriggerConfigurationParam = (
    FileArrivalTriggerConfigurationDict | FileArrivalTriggerConfiguration
)
