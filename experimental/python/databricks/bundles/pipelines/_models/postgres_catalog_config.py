from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.postgres_slot_config import (
    PostgresSlotConfig,
    PostgresSlotConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PostgresCatalogConfig:
    """
    :meta private: [EXPERIMENTAL]

    PG-specific catalog-level configuration parameters
    """

    slot_config: VariableOrOptional[PostgresSlotConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Optional. The Postgres slot configuration to use for logical replication
    """

    @classmethod
    def from_dict(cls, value: "PostgresCatalogConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PostgresCatalogConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class PostgresCatalogConfigDict(TypedDict, total=False):
    """"""

    slot_config: VariableOrOptional[PostgresSlotConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Optional. The Postgres slot configuration to use for logical replication
    """


PostgresCatalogConfigParam = PostgresCatalogConfigDict | PostgresCatalogConfig
