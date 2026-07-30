from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class QueueSettings:
    """"""

    enabled: VariableOr[bool]
    """
    If true, enable queueing for the job. This is a required field.
    """

    @classmethod
    def from_dict(cls, value: "QueueSettingsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "QueueSettingsDict":
        return _transform_to_json_value(self)  # type:ignore


class QueueSettingsDict(TypedDict, total=False):
    """"""

    enabled: VariableOr[bool]
    """
    If true, enable queueing for the job. This is a required field.
    """


QueueSettingsParam = QueueSettingsDict | QueueSettings
