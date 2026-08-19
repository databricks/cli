from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.api_source_connector_config import (
    ApiSourceConnectorConfig,
    ApiSourceConnectorConfigParam,
)
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

    api_source_connector_config: VariableOrOptional[ApiSourceConnectorConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Connector-specific top-level configuration for API Source connectors.
    """

    catalog: VariableOrOptional[SourceCatalogConfig] = None
    """
    [Public Preview] Catalog-level source configuration parameters
    """

    google_ads_config: VariableOrOptional[GoogleAdsConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    @classmethod
    def from_dict(cls, value: "SourceConfigDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SourceConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class SourceConfigDict(TypedDict, total=False):
    """"""

    api_source_connector_config: VariableOrOptional[ApiSourceConnectorConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Connector-specific top-level configuration for API Source connectors.
    """

    catalog: VariableOrOptional[SourceCatalogConfigParam]
    """
    [Public Preview] Catalog-level source configuration parameters
    """

    google_ads_config: VariableOrOptional[GoogleAdsConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """


SourceConfigParam = SourceConfigDict | SourceConfig
