from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.pipelines._models.adlsgen2_info import Adlsgen2Info, Adlsgen2InfoDict, Adlsgen2InfoParam
from databricks.bundles.pipelines._models.aws_attributes import AwsAttributes, AwsAttributesDict, AwsAttributesParam
from databricks.bundles.pipelines._models.cluster_log_conf import ClusterLogConf, ClusterLogConfDict, ClusterLogConfParam
from databricks.bundles.pipelines._models.dbfs_storage_info import DbfsStorageInfo, DbfsStorageInfoDict, DbfsStorageInfoParam
from databricks.bundles.pipelines._models.gcp_attributes import GcpAttributes, GcpAttributesDict, GcpAttributesParam
from databricks.bundles.pipelines._models.gcs_storage_info import GcsStorageInfo, GcsStorageInfoDict, GcsStorageInfoParam
from databricks.bundles.pipelines._models.init_script_info import InitScriptInfo, InitScriptInfoDict, InitScriptInfoParam
from databricks.bundles.pipelines._models.local_file_info import LocalFileInfo, LocalFileInfoDict, LocalFileInfoParam
from databricks.bundles.pipelines._models.log_analytics_info import LogAnalyticsInfo, LogAnalyticsInfoDict, LogAnalyticsInfoParam
from databricks.bundles.pipelines._models.maven_library import MavenLibrary, MavenLibraryDict, MavenLibraryParam
from databricks.bundles.pipelines._models.s3_storage_info import S3StorageInfo, S3StorageInfoDict, S3StorageInfoParam
from databricks.bundles.pipelines._models.volumes_storage_info import VolumesStorageInfo, VolumesStorageInfoDict, VolumesStorageInfoParam
from databricks.bundles.pipelines._models.workspace_storage_info import WorkspaceStorageInfo, WorkspaceStorageInfoDict, WorkspaceStorageInfoParam
from databricks.bundles.pipelines._models.auto_full_refresh_policy import AutoFullRefreshPolicy, AutoFullRefreshPolicyDict, AutoFullRefreshPolicyParam
from databricks.bundles.pipelines._models.confluence_connector_options import ConfluenceConnectorOptions, ConfluenceConnectorOptionsDict, ConfluenceConnectorOptionsParam
from databricks.bundles.pipelines._models.connection_parameters import ConnectionParameters, ConnectionParametersDict, ConnectionParametersParam
from databricks.bundles.pipelines._models.connector_options import ConnectorOptions, ConnectorOptionsDict, ConnectorOptionsParam
from databricks.bundles.pipelines._models.data_staging_options import DataStagingOptions, DataStagingOptionsDict, DataStagingOptionsParam
from databricks.bundles.pipelines._models.event_log_spec import EventLogSpec, EventLogSpecDict, EventLogSpecParam
from databricks.bundles.pipelines._models.file_filter import FileFilter, FileFilterDict, FileFilterParam
from databricks.bundles.pipelines._models.file_ingestion_options import FileIngestionOptions, FileIngestionOptionsDict, FileIngestionOptionsParam
from databricks.bundles.pipelines._models.file_library import FileLibrary, FileLibraryDict, FileLibraryParam
from databricks.bundles.pipelines._models.filters import Filters, FiltersDict, FiltersParam
from databricks.bundles.pipelines._models.google_ads_config import GoogleAdsConfig, GoogleAdsConfigDict, GoogleAdsConfigParam
from databricks.bundles.pipelines._models.google_ads_custom_report_options import GoogleAdsCustomReportOptions, GoogleAdsCustomReportOptionsDict, GoogleAdsCustomReportOptionsParam
from databricks.bundles.pipelines._models.google_ads_options import GoogleAdsOptions, GoogleAdsOptionsDict, GoogleAdsOptionsParam
from databricks.bundles.pipelines._models.google_drive_options import GoogleDriveOptions, GoogleDriveOptionsDict, GoogleDriveOptionsParam
from databricks.bundles.pipelines._models.ingestion_config import IngestionConfig, IngestionConfigDict, IngestionConfigParam
from databricks.bundles.pipelines._models.ingestion_gateway_pipeline_definition import IngestionGatewayPipelineDefinition, IngestionGatewayPipelineDefinitionDict, IngestionGatewayPipelineDefinitionParam
from databricks.bundles.pipelines._models.ingestion_pipeline_definition import IngestionPipelineDefinition, IngestionPipelineDefinitionDict, IngestionPipelineDefinitionParam
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_fanout_options import IngestionPipelineDefinitionFanoutOptions, IngestionPipelineDefinitionFanoutOptionsDict, IngestionPipelineDefinitionFanoutOptionsParam
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_table_specific_config_query_based_connector_config import IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig, IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigDict, IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigParam
from databricks.bundles.pipelines._models.ingestion_pipeline_definition_workday_report_parameters import IngestionPipelineDefinitionWorkdayReportParameters, IngestionPipelineDefinitionWorkdayReportParametersDict, IngestionPipelineDefinitionWorkdayReportParametersParam
from databricks.bundles.pipelines._models.jira_connector_options import JiraConnectorOptions, JiraConnectorOptionsDict, JiraConnectorOptionsParam
from databricks.bundles.pipelines._models.json_transformer_options import JsonTransformerOptions, JsonTransformerOptionsDict, JsonTransformerOptionsParam
from databricks.bundles.pipelines._models.kafka_options import KafkaOptions, KafkaOptionsDict, KafkaOptionsParam
from databricks.bundles.pipelines._models.meta_marketing_options import MetaMarketingOptions, MetaMarketingOptionsDict, MetaMarketingOptionsParam
from databricks.bundles.pipelines._models.meta_marketing_options_meta_marketing_custom_report_options import MetaMarketingOptionsMetaMarketingCustomReportOptions, MetaMarketingOptionsMetaMarketingCustomReportOptionsDict, MetaMarketingOptionsMetaMarketingCustomReportOptionsParam
from databricks.bundles.pipelines._models.notebook_library import NotebookLibrary, NotebookLibraryDict, NotebookLibraryParam
from databricks.bundles.pipelines._models.notifications import Notifications, NotificationsDict, NotificationsParam
from databricks.bundles.pipelines._models.operation_time_window import OperationTimeWindow, OperationTimeWindowDict, OperationTimeWindowParam
from databricks.bundles.pipelines._models.outlook_options import OutlookOptions, OutlookOptionsDict, OutlookOptionsParam
from databricks.bundles.pipelines._models.path_pattern import PathPattern, PathPatternDict, PathPatternParam
from databricks.bundles.pipelines._models.pipeline_cluster import PipelineCluster, PipelineClusterDict, PipelineClusterParam
from databricks.bundles.pipelines._models.pipeline_cluster_autoscale import PipelineClusterAutoscale, PipelineClusterAutoscaleDict, PipelineClusterAutoscaleParam
from databricks.bundles.pipelines._models.pipeline_library import PipelineLibrary, PipelineLibraryDict, PipelineLibraryParam
from databricks.bundles.pipelines._models.pipelines_environment import PipelinesEnvironment, PipelinesEnvironmentDict, PipelinesEnvironmentParam
from databricks.bundles.pipelines._models.postgres_catalog_config import PostgresCatalogConfig, PostgresCatalogConfigDict, PostgresCatalogConfigParam
from databricks.bundles.pipelines._models.postgres_slot_config import PostgresSlotConfig, PostgresSlotConfigDict, PostgresSlotConfigParam
from databricks.bundles.pipelines._models.reddit_ads_options import RedditAdsOptions, RedditAdsOptionsDict, RedditAdsOptionsParam
from databricks.bundles.pipelines._models.reddit_ads_options_reddit_ads_custom_report_options import RedditAdsOptionsRedditAdsCustomReportOptions, RedditAdsOptionsRedditAdsCustomReportOptionsDict, RedditAdsOptionsRedditAdsCustomReportOptionsParam
from databricks.bundles.pipelines._models.report_spec import ReportSpec, ReportSpecDict, ReportSpecParam
from databricks.bundles.pipelines._models.restart_window import RestartWindow, RestartWindowDict, RestartWindowParam
from databricks.bundles.pipelines._models.run_as import RunAs, RunAsDict, RunAsParam
from databricks.bundles.pipelines._models.schema_spec import SchemaSpec, SchemaSpecDict, SchemaSpecParam
from databricks.bundles.pipelines._models.sharepoint_options import SharepointOptions, SharepointOptionsDict, SharepointOptionsParam
from databricks.bundles.pipelines._models.smartsheet_options import SmartsheetOptions, SmartsheetOptionsDict, SmartsheetOptionsParam
from databricks.bundles.pipelines._models.source_catalog_config import SourceCatalogConfig, SourceCatalogConfigDict, SourceCatalogConfigParam
from databricks.bundles.pipelines._models.source_config import SourceConfig, SourceConfigDict, SourceConfigParam
from databricks.bundles.pipelines._models.table_spec import TableSpec, TableSpecDict, TableSpecParam
from databricks.bundles.pipelines._models.table_specific_config import TableSpecificConfig, TableSpecificConfigDict, TableSpecificConfigParam
from databricks.bundles.pipelines._models.tik_tok_ads_options import TikTokAdsOptions, TikTokAdsOptionsDict, TikTokAdsOptionsParam
from databricks.bundles.pipelines._models.tik_tok_ads_options_tik_tok_ads_custom_report_options import TikTokAdsOptionsTikTokAdsCustomReportOptions, TikTokAdsOptionsTikTokAdsCustomReportOptionsDict, TikTokAdsOptionsTikTokAdsCustomReportOptionsParam
from databricks.bundles.pipelines._models.transformer import Transformer, TransformerDict, TransformerParam
from databricks.bundles.pipelines._models.zendesk_support_options import ZendeskSupportOptions, ZendeskSupportOptionsDict, ZendeskSupportOptionsParam
from databricks.bundles.pipelines._models.lifecycle import Lifecycle, LifecycleDict, LifecycleParam
from databricks.bundles.pipelines._models.pipeline import Pipeline, PipelineDict, PipelineParam
from databricks.bundles.pipelines._models.pipeline_permission import PipelinePermission, PipelinePermissionDict, PipelinePermissionParam
from databricks.bundles.pipelines._models.aws_availability import AwsAvailability, AwsAvailabilityParam

from databricks.bundles.pipelines._models.azure_availability import AzureAvailability, AzureAvailabilityParam

from databricks.bundles.pipelines._models.confidential_compute_type import ConfidentialComputeType, ConfidentialComputeTypeParam

from databricks.bundles.pipelines._models.ebs_volume_type import EbsVolumeType, EbsVolumeTypeParam

from databricks.bundles.pipelines._models.gcp_availability import GcpAvailability, GcpAvailabilityParam

from databricks.bundles.pipelines._models.connector_type import ConnectorType, ConnectorTypeParam

from databricks.bundles.pipelines._models.day_of_week import DayOfWeek, DayOfWeekParam

from databricks.bundles.pipelines._models.file_ingestion_options_file_format import FileIngestionOptionsFileFormat, FileIngestionOptionsFileFormatParam

from databricks.bundles.pipelines._models.file_ingestion_options_schema_evolution_mode import FileIngestionOptionsSchemaEvolutionMode, FileIngestionOptionsSchemaEvolutionModeParam

from databricks.bundles.pipelines._models.google_drive_options_google_drive_entity_type import GoogleDriveOptionsGoogleDriveEntityType, GoogleDriveOptionsGoogleDriveEntityTypeParam

from databricks.bundles.pipelines._models.outlook_attachment_mode import OutlookAttachmentMode, OutlookAttachmentModeParam

from databricks.bundles.pipelines._models.outlook_body_format import OutlookBodyFormat, OutlookBodyFormatParam

from databricks.bundles.pipelines._models.pipeline_cluster_autoscale_mode import PipelineClusterAutoscaleMode, PipelineClusterAutoscaleModeParam

from databricks.bundles.pipelines._models.pipeline_permission_level import PipelinePermissionLevel, PipelinePermissionLevelParam

from databricks.bundles.pipelines._models.sharepoint_options_sharepoint_entity_type import SharepointOptionsSharepointEntityType, SharepointOptionsSharepointEntityTypeParam

from databricks.bundles.pipelines._models.table_specific_config_scd_type import TableSpecificConfigScdType, TableSpecificConfigScdTypeParam

from databricks.bundles.pipelines._models.tik_tok_ads_options_tik_tok_data_level import TikTokAdsOptionsTikTokDataLevel, TikTokAdsOptionsTikTokDataLevelParam

from databricks.bundles.pipelines._models.tik_tok_ads_options_tik_tok_report_type import TikTokAdsOptionsTikTokReportType, TikTokAdsOptionsTikTokReportTypeParam

from databricks.bundles.pipelines._models.transformer_format import TransformerFormat, TransformerFormatParam


if TYPE_CHECKING:
    from typing_extensions import Self

@dataclass(kw_only=True)
class AzureAttributes:
    """
    Attributes set during cluster creation which are related to Microsoft Azure.
    """
    availability: VariableOrOptional[AzureAvailability] = None
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    Note: If `first_on_demand` is zero, this availability
    type will be used for the entire cluster.
    """

    capacity_reservation_group: VariableOrOptional[str] = None
    """
    The Azure capacity reservation group resource ID to use for launching VMs.
    When specified, VMs will be launched using the provided capacity reservation.
    
    Capacity reservations can only be specified when the workspace uses injected vnet (i.e. customer defined vnet not
    managed by databricks). Ensure the databricks-login-prod Enterprise Application is granted the following four permissions:
    1. Microsoft.Compute/capacityReservationGroups/read
    2. Microsoft.Compute/capacityReservationGroups/deploy/action
    3. Microsoft.Compute/capacityReservationGroups/capacityReservations/read
    4. Microsoft.Compute/capacityReservationGroups/capacityReservations/deploy/action
    
    Format: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/capacityReservationGroups/{capacityReservationGroupName}`
    """

    first_on_demand: VariableOrOptional[int] = None
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    This value should be greater than 0, to make sure the cluster driver node is placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    log_analytics_info: VariableOrOptional[LogAnalyticsInfo] = None
    """
    Defines values necessary to configure and run Azure Log Analytics agent
    """

    spot_bid_max_price: VariableOrOptional[float] = None
    """
    The max bid price to be used for Azure spot instances.
    The Max price for the bid cannot be higher than the on-demand price of the instance.
    If not specified, the default value is -1, which specifies that the instance cannot be evicted
    on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
    """

    @classmethod
    def from_dict(cls, value: 'AzureAttributesDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'AzureAttributesDict':
        return _transform_to_json_value(self) # type:ignore



class AzureAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[AzureAvailabilityParam]
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    Note: If `first_on_demand` is zero, this availability
    type will be used for the entire cluster.
    """

    capacity_reservation_group: VariableOrOptional[str]
    """
    The Azure capacity reservation group resource ID to use for launching VMs.
    When specified, VMs will be launched using the provided capacity reservation.
    
    Capacity reservations can only be specified when the workspace uses injected vnet (i.e. customer defined vnet not
    managed by databricks). Ensure the databricks-login-prod Enterprise Application is granted the following four permissions:
    1. Microsoft.Compute/capacityReservationGroups/read
    2. Microsoft.Compute/capacityReservationGroups/deploy/action
    3. Microsoft.Compute/capacityReservationGroups/capacityReservations/read
    4. Microsoft.Compute/capacityReservationGroups/capacityReservations/deploy/action
    
    Format: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/capacityReservationGroups/{capacityReservationGroupName}`
    """

    first_on_demand: VariableOrOptional[int]
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    This value should be greater than 0, to make sure the cluster driver node is placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    log_analytics_info: VariableOrOptional[LogAnalyticsInfoParam]
    """
    Defines values necessary to configure and run Azure Log Analytics agent
    """

    spot_bid_max_price: VariableOrOptional[float]
    """
    The max bid price to be used for Azure spot instances.
    The Max price for the bid cannot be higher than the on-demand price of the instance.
    If not specified, the default value is -1, which specifies that the instance cannot be evicted
    on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
    """


AzureAttributesParam = AzureAttributesDict | AzureAttributes
