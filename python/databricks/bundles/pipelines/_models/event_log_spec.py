from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class EventLogSpec:
    """
    Configurable event log parameters.
    """

    catalog: VariableOrOptional[str] = None
    """
    The UC catalog the event log is published under.
    """

    name: VariableOrOptional[str] = None
    """
    The name the event log is published to in UC.
    """

    schema: VariableOrOptional[str] = None
    """
    The UC schema the event log is published under.
    """

    @classmethod
    def from_dict(cls, value: "EventLogSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EventLogSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class EventLogSpecDict(TypedDict, total=False):
    """"""

    catalog: VariableOrOptional[str]
    """
    The UC catalog the event log is published under.
    """

    name: VariableOrOptional[str]
    """
    The name the event log is published to in UC.
    """

    schema: VariableOrOptional[str]
    """
    The UC schema the event log is published under.
    """


EventLogSpecParam = EventLogSpecDict | EventLogSpec
