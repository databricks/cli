from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.pipelines._models.adlsgen2_info import Adlsgen2Info, Adlsgen2InfoDict, Adlsgen2InfoParam
from databricks.bundles.pipelines._models.aws_attributes import AwsAttributes, AwsAttributesDict, AwsAttributesParam
from databricks.bundles.pipelines._models.azure_attributes import AzureAttributes, AzureAttributesDict, AzureAttributesParam
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
class Pipeline(Resource):
    """"""

    allow_duplicate_names: VariableOrOptional[bool] = None
    """
    If false, deployment will fail if name conflicts with that of another pipeline.
    """

    budget_policy_id: VariableOrOptional[str] = None
    """
    [Public Preview] Budget policy of this pipeline.
    """

    catalog: VariableOrOptional[str] = None
    """
    A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
    """

    channel: VariableOrOptional[str] = None
    """
    SDP Release Channel that specifies which version to use.
    """

    clusters: VariableOrList[PipelineCluster] = field(default_factory=list)
    """
    Cluster settings for this pipeline deployment.
    """

    configuration: VariableOrDict[str] = field(default_factory=dict)
    """
    String-String configuration for this pipeline execution.
    """

    continuous: VariableOrOptional[bool] = None
    """
    Whether the pipeline is continuous or triggered. This replaces `trigger`.
    """

    development: VariableOrOptional[bool] = None
    """
    Whether the pipeline is in Development mode. Defaults to false.
    """

    edition: VariableOrOptional[str] = None
    """
    Pipeline product edition.
    """

    environment: VariableOrOptional[PipelinesEnvironment] = None
    """
    [Public Preview] Environment specification for this pipeline used to install dependencies.
    """

    event_log: VariableOrOptional[EventLogSpec] = None
    """
    Event log configuration for this pipeline
    """

    filters: VariableOrOptional[Filters] = None
    """
    Filters on which Pipeline packages to include in the deployed graph.
    """

    gateway_definition: VariableOrOptional[IngestionGatewayPipelineDefinition] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The definition of a gateway pipeline to support change data capture.
    """

    id: VariableOrOptional[str] = None
    """
    Unique identifier for this pipeline.
    """

    ingestion_definition: VariableOrOptional[IngestionPipelineDefinition] = None
    """
    [Public Preview] The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'schema', 'target', or 'catalog' settings.
    """

    libraries: VariableOrList[PipelineLibrary] = field(default_factory=list)
    """
    Libraries or code needed by this deployment.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    name: VariableOrOptional[str] = None
    """
    Friendly identifier for this pipeline.
    """

    notifications: VariableOrList[Notifications] = field(default_factory=list)
    """
    List of notification settings for this pipeline.
    """

    parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    [Beta] Key/value map of default parameters to use for pipeline execution.
    Maximum total size: 10k characters (JSON format)
    """

    permissions: VariableOrList[PipelinePermission] = field(default_factory=list)
    """
    The permissions to apply to this resource.
    """

    photon: VariableOrOptional[bool] = None
    """
    Whether Photon is enabled for this pipeline.
    """

    restart_window: VariableOrOptional[RestartWindow] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Restart window of this pipeline.
    """

    root_path: VariableOrOptional[str] = None
    """
    [Public Preview] Root path for this pipeline.
    This is used as the root directory when editing the pipeline in the Databricks user interface and it is
    added to sys.path when executing Python sources during pipeline execution.
    """

    run_as: VariableOrOptional[RunAs] = None
    """
    Write-only setting, available only in Create/Update calls. Specifies the user or service principal that the pipeline runs as. If not specified, the pipeline runs as the user who created the pipeline.
    
    Only `user_name` or `service_principal_name` can be specified. If both are specified, an error is thrown.
    """

    schema: VariableOrOptional[str] = None
    """
    The default schema (database) where tables are read from or published to.
    """

    serverless: VariableOrOptional[bool] = None
    """
    Whether serverless compute is enabled for this pipeline.
    """

    serverless_compute_id: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Serverless compute ID specified by the user for serverless pipelines.
    """

    storage: VariableOrOptional[str] = None
    """
    DBFS root directory for storing checkpoints and tables.
    """

    tags: VariableOrDict[str] = field(default_factory=dict)
    """
    A map of tags associated with the pipeline.
    These are forwarded to the cluster as cluster tags, and are therefore subject to the same limitations.
    A maximum of 25 tags can be added to the pipeline.
    """

    target: VariableOrOptional[str] = None
    """
    [DEPRECATED] Target schema (database) to add tables in this pipeline to. Exactly one of `schema` or `target` must be specified. To publish to Unity Catalog, also specify `catalog`. This legacy field is deprecated for pipeline creation in favor of the `schema` field.
    """

    usage_policy_id: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Usage policy of this pipeline.
    """

    @classmethod
    def from_dict(cls, value: 'PipelineDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'PipelineDict':
        return _transform_to_json_value(self) # type:ignore



class PipelineDict(TypedDict, total=False):
    """"""

    allow_duplicate_names: VariableOrOptional[bool]
    """
    If false, deployment will fail if name conflicts with that of another pipeline.
    """

    budget_policy_id: VariableOrOptional[str]
    """
    [Public Preview] Budget policy of this pipeline.
    """

    catalog: VariableOrOptional[str]
    """
    A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
    """

    channel: VariableOrOptional[str]
    """
    SDP Release Channel that specifies which version to use.
    """

    clusters: VariableOrList[PipelineClusterParam]
    """
    Cluster settings for this pipeline deployment.
    """

    configuration: VariableOrDict[str]
    """
    String-String configuration for this pipeline execution.
    """

    continuous: VariableOrOptional[bool]
    """
    Whether the pipeline is continuous or triggered. This replaces `trigger`.
    """

    development: VariableOrOptional[bool]
    """
    Whether the pipeline is in Development mode. Defaults to false.
    """

    edition: VariableOrOptional[str]
    """
    Pipeline product edition.
    """

    environment: VariableOrOptional[PipelinesEnvironmentParam]
    """
    [Public Preview] Environment specification for this pipeline used to install dependencies.
    """

    event_log: VariableOrOptional[EventLogSpecParam]
    """
    Event log configuration for this pipeline
    """

    filters: VariableOrOptional[FiltersParam]
    """
    Filters on which Pipeline packages to include in the deployed graph.
    """

    gateway_definition: VariableOrOptional[IngestionGatewayPipelineDefinitionParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The definition of a gateway pipeline to support change data capture.
    """

    id: VariableOrOptional[str]
    """
    Unique identifier for this pipeline.
    """

    ingestion_definition: VariableOrOptional[IngestionPipelineDefinitionParam]
    """
    [Public Preview] The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'schema', 'target', or 'catalog' settings.
    """

    libraries: VariableOrList[PipelineLibraryParam]
    """
    Libraries or code needed by this deployment.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    name: VariableOrOptional[str]
    """
    Friendly identifier for this pipeline.
    """

    notifications: VariableOrList[NotificationsParam]
    """
    List of notification settings for this pipeline.
    """

    parameters: VariableOrDict[str]
    """
    [Beta] Key/value map of default parameters to use for pipeline execution.
    Maximum total size: 10k characters (JSON format)
    """

    permissions: VariableOrList[PipelinePermissionParam]
    """
    The permissions to apply to this resource.
    """

    photon: VariableOrOptional[bool]
    """
    Whether Photon is enabled for this pipeline.
    """

    restart_window: VariableOrOptional[RestartWindowParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Restart window of this pipeline.
    """

    root_path: VariableOrOptional[str]
    """
    [Public Preview] Root path for this pipeline.
    This is used as the root directory when editing the pipeline in the Databricks user interface and it is
    added to sys.path when executing Python sources during pipeline execution.
    """

    run_as: VariableOrOptional[RunAsParam]
    """
    Write-only setting, available only in Create/Update calls. Specifies the user or service principal that the pipeline runs as. If not specified, the pipeline runs as the user who created the pipeline.
    
    Only `user_name` or `service_principal_name` can be specified. If both are specified, an error is thrown.
    """

    schema: VariableOrOptional[str]
    """
    The default schema (database) where tables are read from or published to.
    """

    serverless: VariableOrOptional[bool]
    """
    Whether serverless compute is enabled for this pipeline.
    """

    serverless_compute_id: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Serverless compute ID specified by the user for serverless pipelines.
    """

    storage: VariableOrOptional[str]
    """
    DBFS root directory for storing checkpoints and tables.
    """

    tags: VariableOrDict[str]
    """
    A map of tags associated with the pipeline.
    These are forwarded to the cluster as cluster tags, and are therefore subject to the same limitations.
    A maximum of 25 tags can be added to the pipeline.
    """

    target: VariableOrOptional[str]
    """
    [DEPRECATED] Target schema (database) to add tables in this pipeline to. Exactly one of `schema` or `target` must be specified. To publish to Unity Catalog, also specify `catalog`. This legacy field is deprecated for pipeline creation in favor of the `schema` field.
    """

    usage_policy_id: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Usage policy of this pipeline.
    """


PipelineParam = PipelineDict | Pipeline
