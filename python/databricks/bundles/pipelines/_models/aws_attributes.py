from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.pipelines._models.adlsgen2_info import Adlsgen2Info, Adlsgen2InfoDict, Adlsgen2InfoParam
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
class AwsAttributes:
    """
    Attributes set during cluster creation which are related to Amazon Web Services.
    """
    availability: VariableOrOptional[AwsAvailability] = None
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    
    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    ebs_volume_count: VariableOrOptional[int] = None
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int] = None
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeType] = None
    """
    The type of EBS volumes that will be launched with this cluster.
    """

    first_on_demand: VariableOrOptional[int] = None
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str] = None
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int] = None
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str] = None
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, the zone "auto" will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """

    @classmethod
    def from_dict(cls, value: 'AwsAttributesDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'AwsAttributesDict':
        return _transform_to_json_value(self) # type:ignore



class AwsAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[AwsAvailabilityParam]
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    
    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    ebs_volume_count: VariableOrOptional[int]
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int]
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int]
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int]
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeTypeParam]
    """
    The type of EBS volumes that will be launched with this cluster.
    """

    first_on_demand: VariableOrOptional[int]
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str]
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int]
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str]
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, the zone "auto" will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """


AwsAttributesParam = AwsAttributesDict | AwsAttributes
