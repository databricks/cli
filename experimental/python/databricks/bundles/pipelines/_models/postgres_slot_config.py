from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PostgresSlotConfig:
    """
    :meta private: [EXPERIMENTAL]

    PostgresSlotConfig contains the configuration for a Postgres logical replication slot
    """

    publication_name: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    The name of the publication to use for the Postgres source
    """

    slot_name: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    The name of the logical replication slot to use for the Postgres source
    """

    @classmethod
    def from_dict(cls, value: "PostgresSlotConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PostgresSlotConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class PostgresSlotConfigDict(TypedDict, total=False):
    """"""

    publication_name: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    The name of the publication to use for the Postgres source
    """

    slot_name: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    The name of the logical replication slot to use for the Postgres source
    """


PostgresSlotConfigParam = PostgresSlotConfigDict | PostgresSlotConfig
