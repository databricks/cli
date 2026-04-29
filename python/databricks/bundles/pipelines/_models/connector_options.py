from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.google_ads_options import (
    GoogleAdsOptions,
    GoogleAdsOptionsParam,
)
from databricks.bundles.pipelines._models.google_drive_options import (
    GoogleDriveOptions,
    GoogleDriveOptionsParam,
)
from databricks.bundles.pipelines._models.jira_connector_options import (
    JiraConnectorOptions,
    JiraConnectorOptionsParam,
)
from databricks.bundles.pipelines._models.outlook_options import (
    OutlookOptions,
    OutlookOptionsParam,
)
from databricks.bundles.pipelines._models.sharepoint_options import (
    SharepointOptions,
    SharepointOptionsParam,
)
from databricks.bundles.pipelines._models.smartsheet_options import (
    SmartsheetOptions,
    SmartsheetOptionsParam,
)
from databricks.bundles.pipelines._models.tik_tok_ads_options import (
    TikTokAdsOptions,
    TikTokAdsOptionsParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ConnectorOptions:
    """
    Wrapper message for source-specific options to support multiple connector types
    """

    gdrive_options: VariableOrOptional[GoogleDriveOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    google_ads_options: VariableOrOptional[GoogleAdsOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Google Ads specific options for ingestion (object-level).
    When set, these values override the corresponding fields in GoogleAdsConfig
    (source_configurations).
    """

    jira_options: VariableOrOptional[JiraConnectorOptions] = None
    """
    Jira specific options for ingestion
    """

    outlook_options: VariableOrOptional[OutlookOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Outlook specific options for ingestion
    """

    sharepoint_options: VariableOrOptional[SharepointOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    smartsheet_options: VariableOrOptional[SmartsheetOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Smartsheet specific options for ingestion
    """

    tiktok_ads_options: VariableOrOptional[TikTokAdsOptions] = None
    """
    :meta private: [EXPERIMENTAL]
    
    TikTok Ads specific options for ingestion
    """

    @classmethod
    def from_dict(cls, value: "ConnectorOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ConnectorOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class ConnectorOptionsDict(TypedDict, total=False):
    """"""

    gdrive_options: VariableOrOptional[GoogleDriveOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    """

    google_ads_options: VariableOrOptional[GoogleAdsOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Google Ads specific options for ingestion (object-level).
    When set, these values override the corresponding fields in GoogleAdsConfig
    (source_configurations).
    """

    jira_options: VariableOrOptional[JiraConnectorOptionsParam]
    """
    Jira specific options for ingestion
    """

    outlook_options: VariableOrOptional[OutlookOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Outlook specific options for ingestion
    """

    sharepoint_options: VariableOrOptional[SharepointOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    """

    smartsheet_options: VariableOrOptional[SmartsheetOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Smartsheet specific options for ingestion
    """

    tiktok_ads_options: VariableOrOptional[TikTokAdsOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    
    TikTok Ads specific options for ingestion
    """


ConnectorOptionsParam = ConnectorOptionsDict | ConnectorOptions
