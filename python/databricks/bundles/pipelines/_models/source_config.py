from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.google_ads_config import (
    GoogleAdsConfig,
    GoogleAdsConfigParam,
)
from databricks.bundles.pipelines._models.source_catalog_config import (
    SourceCatalogConfig,
    SourceCatalogConfigParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SourceConfig:
    """"""

    catalog: VariableOrOptional[SourceCatalogConfig] = None
    """
    Catalog-level source configuration parameters
    """

    google_ads_config: VariableOrOptional[GoogleAdsConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    @classmethod
    def from_dict(cls, value: "SourceConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SourceConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class SourceConfigDict(TypedDict, total=False):
    """"""

    catalog: VariableOrOptional[SourceCatalogConfigParam]
    """
    Catalog-level source configuration parameters
    """

    google_ads_config: VariableOrOptional[GoogleAdsConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    """


SourceConfigParam = SourceConfigDict | SourceConfig
