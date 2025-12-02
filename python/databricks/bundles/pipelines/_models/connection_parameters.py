from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ConnectionParameters:
    """
    :meta private: [EXPERIMENTAL]
    """

    source_catalog: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Source catalog for initial connection.
    This is necessary for schema exploration in some database systems like Oracle, and optional but nice-to-have
    in some other database systems like Postgres.
    For Oracle databases, this maps to a service name.
    """

    @classmethod
    def from_dict(cls, value: "ConnectionParametersDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ConnectionParametersDict":
        return _transform_to_json_value(self)  # type:ignore


class ConnectionParametersDict(TypedDict, total=False):
    """"""

    source_catalog: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Source catalog for initial connection.
    This is necessary for schema exploration in some database systems like Oracle, and optional but nice-to-have
    in some other database systems like Postgres.
    For Oracle databases, this maps to a service name.
    """


ConnectionParametersParam = ConnectionParametersDict | ConnectionParameters
