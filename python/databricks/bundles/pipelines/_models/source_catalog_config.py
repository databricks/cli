from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.postgres_catalog_config import (
    PostgresCatalogConfig,
    PostgresCatalogConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SourceCatalogConfig:
    """
    :meta private: [EXPERIMENTAL]

    SourceCatalogConfig contains catalog-level custom configuration parameters for each source
    """

    postgres: VariableOrOptional[PostgresCatalogConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Postgres-specific catalog-level configuration parameters
    """

    source_catalog: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Source catalog name
    """

    @classmethod
    def from_dict(cls, value: "SourceCatalogConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SourceCatalogConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class SourceCatalogConfigDict(TypedDict, total=False):
    """"""

    postgres: VariableOrOptional[PostgresCatalogConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Postgres-specific catalog-level configuration parameters
    """

    source_catalog: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Source catalog name
    """


SourceCatalogConfigParam = SourceCatalogConfigDict | SourceCatalogConfig
